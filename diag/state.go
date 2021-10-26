package diag

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"strconv"
	"strings"
)

func (context *diagContext) getOrAddState(namespace, kind, name string) *entityState {
	eName := entityName{
		namespace: namespace,
		kind:      kind,
		name:      name,
	}
	state, found := context.statesByName[eName]
	if !found {
		state = newState(eName)
		context.statesByName[eName] = state
	}
	return state
}

func (context *diagContext) addEventState(namespace, kind, name string) *eventState {
	eName := entityName{
		namespace: namespace,
		kind:      kind,
		name:      name,
	}
	eventsOfEntity, exists := context.eventsByName[eName]
	if !exists {
		eventsOfEntity = []*eventState{}
	}
	evState := &eventState{
		name: entityName{
			namespace: namespace,
			kind:      kind,
			name:      name,
		},
	}
	context.eventsByName[eName] = append(eventsOfEntity, evState)
	return evState
}

func (state *entityState) checkStatuses(pod *v1.Pod, statuses []v1.ContainerStatus, initContainers bool, context *diagContext) bool {
	subTitle := ""
	if initContainers {
		subTitle = " (init)"
	}
	anyMessage := false
	for _, containerStatus := range statuses {
		anyMessageForContainer := false
		stateTerminated := containerStatus.State.Terminated
		stateWaiting := containerStatus.State.Waiting
		if stateTerminated != nil {
			terminatedReason := stateTerminated.Reason
			if terminatedReason != "Completed" {
				state.appendMessage("%v%v terminated due to %v (exit code %v)", containerStatus.Name, subTitle, terminatedReason, stateTerminated.ExitCode)
				anyMessageForContainer = true
			}
		}
		if stateWaiting != nil && stateWaiting.Reason != "ContainerCreating" && stateWaiting.Reason != "PodInitializing" {
			message := stateWaiting.Message
			detailsIndex := strings.Index(message, "restarting failed container")
			if detailsIndex >= 0 {
				message = message[:(detailsIndex + 27)]
			}
			state.appendMessage("%v%v still waiting due to %v: %v", containerStatus.Name, subTitle, stateWaiting.Reason, wrapTemporal(message))
			anyMessageForContainer = true
		}
		if containerStatus.RestartCount > context.config.PodRestartGraceCount {
			stateTerminated = containerStatus.LastTerminationState.Terminated
			if stateTerminated != nil {
				state.appendMessage("%v%v had restarted %v times, last exit due to %v (exit code %v)", containerStatus.Name, subTitle, wrapTemporal(containerStatus.RestartCount), stateTerminated.Reason, stateTerminated.ExitCode)
			} else {
				state.appendMessage("%v%v had restarted %v times", containerStatus.Name, subTitle, wrapTemporal(containerStatus.RestartCount))
			}
			anyMessageForContainer = true
		}
		if anyMessageForContainer && context.client != nil {
			logs, err := context.client.GetPodLogs(pod.Namespace, pod.Name, containerStatus.Name)
			if err != nil {
				log.Errorf("failed to get logs of %v/%v/%v: %v", pod.Namespace, pod.Name, containerStatus.Name, err)
			} else {
				logs = strings.TrimSpace(logs)
				if logs != "" {
					state.logsCollections[containerStatus.Name] = logs
				}
			}
		}
		anyMessage = anyMessage || anyMessageForContainer
	}
	return anyMessage
}

func (context *diagContext) podState(pod *v1.Pod) (state *entityState, err error) {
	state = context.getOrAddState(pod.Namespace, "Pod", pod.Name)

	podPhase := pod.Status.Phase
	if podPhase == v1.PodSucceeded {
		return
	}

	baselineTime := pod.ObjectMeta.CreationTimestamp.Time
	if pod.Status.StartTime != nil {
		baselineTime = pod.Status.StartTime.Time
	}
	sinceCreation := context.now.Sub(baselineTime).Seconds()
	if sinceCreation < context.config.PodCreationGracePeriodSeconds {
		return
	}

	statusReason := pod.Status.Reason
	if statusReason != "" {
		statusMessage := strings.TrimSpace(pod.Status.Message)
		if statusReason == "Evicted" {
			statusMessage = wrapTemporal(formatUnitsSize(statusMessage))
		}
		state.appendMessage("Pod is in %v phase due to %v: %v", podPhase, statusReason, statusMessage)
	} else if pod.DeletionTimestamp != nil {
		deletionTime := (*pod.DeletionTimestamp).Time
		if context.now.Sub(deletionTime).Seconds() > float64(valueOrDefault(pod.DeletionGracePeriodSeconds, context.config.PodTerminationGracePeriodSeconds)) {
			suffix := ""
			if pod.DeletionGracePeriodSeconds != nil && *pod.DeletionGracePeriodSeconds != 0 {
				suffix = fmt.Sprintf(" (deletion grace is %v sec)", *pod.DeletionGracePeriodSeconds)
			}
			state.appendMessage("Pod is Terminating since %v%v", wrapTemporal(formatDuration(deletionTime, context.now)), suffix)
		}
	} else if podPhase != v1.PodRunning {
		state.appendMessage("Pod is in %v phase", podPhase)
	}

	anyStatusMessage := state.checkStatuses(pod, pod.Status.ContainerStatuses, false, context)
	anyStatusMessage = anyStatusMessage || state.checkStatuses(pod, pod.Status.InitContainerStatuses, true, context)

	if !anyStatusMessage && pod.DeletionTimestamp == nil {
		for _, condition := range pod.Status.Conditions {
			if condition.Status != "True" {
				state.appendMessage(
					"%v: %v (last transition: %v)",
					splitToWords(condition.Reason),
					condition.Message,
					wrapTemporal(formatDuration(condition.LastTransitionTime.Time, context.now)),
				)
			}
		}
	}

	return
}

func (context *diagContext) eventState(event *v1.Event) (state *eventState, err error) {

	state = context.addEventState(event.InvolvedObject.Namespace, event.InvolvedObject.Kind, event.InvolvedObject.Name)

	state.timestamp = event.EventTime.Time
	if state.timestamp.IsZero() {
		state.timestamp = event.FirstTimestamp.Time
	}

	if event.Type == "Normal" {
		return state, nil
	}

	source := event.Source.Component
	if source == "" {
		source = event.ReportingController
	}

	lastTimestamp := state.timestamp
	count := int32(1)
	if event.Series != nil {
		lastTimestamp = event.Series.LastObservedTime.Time
		count = event.Series.Count
	} else if event.Count > 1 {
		lastTimestamp = event.LastTimestamp.Time
		count = event.Count
	}

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf(
		"Event by %v: %v ",
		source,
		event.Reason,
	))
	if count > 1 {
		builder.WriteString(fmt.Sprintf("x%v ", wrapTemporal(count)))
	}
	builder.WriteString(fmt.Sprintf(
		"since %v (last seen %v)",
		wrapTemporal(formatTime(state.timestamp, context.config.TimeFormat, context.config.Locale)),
		wrapTemporal(formatDuration(lastTimestamp, context.now)),
	))

	if event.Message != "" {
		var lines []string
		for _, line := range strings.Split(event.Message, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			builder.WriteString(":\n\t")
		}
		builder.WriteString(strings.Join(lines, "\n\t"))
	}

	state.message = builder.String()
	return state, nil
}

func (context *diagContext) nodeState(node *v1.Node, forceCheckResources bool) (state *entityState, err error) {
	state = context.getOrAddState(node.Namespace, "Node", node.Name)

	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case "Ready":
			if condition.Status == "True" {
				continue
			}
		default:
			if condition.Status == "False" {
				continue
			}
		}
		state.appendMessage(
			"%v: %v (last transition: %v)",
			splitToWords(condition.Reason),
			formatUnitsSize(condition.Message),
			wrapTemporal(formatDuration(condition.LastTransitionTime.Time, context.now)),
		)
	}

	if !state.isHealthy() && !forceCheckResources {
		return
	}

	state.appendMessage(formatResourceUsage(
		node.Status.Allocatable.Cpu().MilliValue(),
		node.Status.Capacity.Cpu().MilliValue(),
		"CPU", context.config.NodeResourceUsageThreshold,
	))

	state.appendMessage(formatResourceUsage(
		node.Status.Allocatable.Memory().Value(),
		node.Status.Capacity.Memory().Value(),
		"Memory", context.config.NodeResourceUsageThreshold,
	))

	state.appendMessage(formatResourceUsage(
		node.Status.Allocatable.StorageEphemeral().Value(),
		node.Status.Capacity.StorageEphemeral().Value(),
		"Ephemeral Storage", context.config.NodeResourceUsageThreshold,
	))

	return
}

func (context *diagContext) replicaSetState(replicaSet *v12.ReplicaSet) (state *entityState, err error) {
	state = context.getOrAddState(replicaSet.Namespace, "ReplicaSet", replicaSet.Name)

	specDesiredReplicas := replicaSet.Spec.Replicas
	var desiredReplicas int
	if specDesiredReplicas == nil {
		desiredReplicasAnnotation, found := replicaSet.ObjectMeta.Annotations["deployment.kubernetes.io/desired-replicas"]
		if found {
			desiredReplicas, err = strconv.Atoi(desiredReplicasAnnotation)
			if err != nil {
				log.Errorf("Failed to parse desired replicas annotation value '%v': %v", desiredReplicasAnnotation, err)
				err = nil
				desiredReplicas = 1
			}
		} else {
			desiredReplicas = 1
		}
	} else {
		desiredReplicas = int(*specDesiredReplicas)
	}
	if desiredReplicas == 0 {
		return
	}

	currentReplicas := int(replicaSet.Status.Replicas)
	if currentReplicas >= desiredReplicas {
		return
	}

	for _, condition := range replicaSet.Status.Conditions {
		state.appendMessage(
			"%v: %v (last transition: %v)",
			splitToWords(condition.Reason),
			formatUnitsSize(condition.Message),
			wrapTemporal(formatDuration(condition.LastTransitionTime.Time, context.now)),
		)
	}
	return
}
