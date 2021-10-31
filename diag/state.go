package diag

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sort"
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

func (context *diagContext) addEventState(eName entityName) *eventState {
	eventsOfEntity, exists := context.eventsByName[eName]
	if !exists {
		eventsOfEntity = []*eventState{}
	}
	evState := &eventState{
		name: eName,
	}
	context.eventsByName[eName] = append(eventsOfEntity, evState)
	return evState
}

var ignoreWaitingReasons = map[string]bool{
	"CrashLoopBackOff":  true,
	"Completed":         true,
	"ContainerCreating": true,
	"PodInitializing":   true,
}

func (state *entityState) checkContainerStatuses(pod *v1.Pod, context *diagContext) {

	var waitingToCreate []string
	var waitingToInitialize []string
	anyRunProblems := false

	subTitle := " (init)"
	allInitContainersHealthy := true
	for _, containerStatus := range pod.Status.InitContainerStatuses {
		runProblems, containerWaitingToCreate, containerWaitingToInitialize := state.checkContainerStatus(pod, containerStatus, subTitle, context)
		if containerWaitingToCreate {
			waitingToCreate = append(waitingToCreate, containerStatus.Name+subTitle)
		} else if containerWaitingToInitialize {
			waitingToInitialize = append(waitingToInitialize, containerStatus.Name+subTitle)
		}
		if runProblems {
			allInitContainersHealthy = false
			anyRunProblems = true
		}
	}

	if allInitContainersHealthy {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			runProblems, containerWaitingToCreate, containerWaitingToInitialize := state.checkContainerStatus(pod, containerStatus, "", context)
			if containerWaitingToCreate {
				waitingToCreate = append(waitingToCreate, containerStatus.Name)
			} else if containerWaitingToInitialize {
				waitingToInitialize = append(waitingToInitialize, containerStatus.Name)
			}
			anyRunProblems = anyRunProblems || runProblems
		}
	}

	var pending *[]string
	pendingVerb := ""
	if len(waitingToCreate) > 0 {
		pendingVerb = "creating"
		pending = &waitingToCreate
	} else if len(waitingToInitialize) > 0 {
		pendingVerb = "initializing"
		pending = &waitingToInitialize
	}

	if pending != nil {
		sort.Strings(*pending)
		state.appendMessage(
			"%v still %v [ %v ] (since %v)",
			wrapTemporal(formatPlural(len(*pending), "One container is", "containers are")),
			pendingVerb,
			strings.Join(*pending, ", "),
			wrapTemporal(formatDuration(pod.CreationTimestamp.Time, context.now)),
		)
	}

	if pod.Status.Phase != v1.PodRunning && !anyRunProblems && len(state.messages) == 0 {
		anyConditionMessage := false
		for _, condition := range pod.Status.Conditions {
			if condition.Status != "True" {
				anyConditionMessage = true
				state.appendMessage(
					"%v: %v (last transition: %v)",
					splitToWords(condition.Reason),
					condition.Message,
					wrapTemporal(formatDuration(condition.LastTransitionTime.Time, context.now)),
				)
			}
		}
		if !anyConditionMessage {
			sinceCreation := context.now.Sub(pod.CreationTimestamp.Time).Seconds()
			if pod.Status.Phase != v1.PodPending || sinceCreation >= context.config.PodStartingGracePeriodSeconds {
				state.appendMessage("Pod is in %v phase (since %v)", pod.Status.Phase, wrapTemporal(formatDuration(pod.CreationTimestamp.Time, context.now)))
			}
		}
	}
}

func (state *entityState) checkContainerStatus(pod *v1.Pod, containerStatus v1.ContainerStatus, subTitle string, context *diagContext) (runProblems bool, waitingToCreate bool, waitingToInitialize bool) {
	shouldCollectLogs := false
	title := fmt.Sprintf("Container %v%v", containerStatus.Name, subTitle)
	stateTerminated := containerStatus.State.Terminated
	stateWaiting := containerStatus.State.Waiting
	if stateTerminated != nil {
		terminatedReason := stateTerminated.Reason
		if terminatedReason != "Completed" {
			runProblems = true
			sinceTerminated := context.now.Sub(stateTerminated.FinishedAt.Time).Seconds()
			if sinceTerminated >= float64(context.config.PodTerminationGracePeriodSeconds) {
				state.appendMessage("%v%v terminated due to %v (exit code %v)", containerStatus.Name, subTitle, terminatedReason, stateTerminated.ExitCode)
			}
		}
	}

	if stateWaiting != nil {
		runProblems = true
		sinceCreation := context.now.Sub(pod.CreationTimestamp.Time).Seconds()
		startingGracePassed := sinceCreation >= context.config.PodStartingGracePeriodSeconds
		if stateWaiting.Reason == "ContainerCreating" && startingGracePassed {
			waitingToCreate = true
		} else if stateWaiting.Reason == "PodInitializing" && startingGracePassed {
			waitingToInitialize = true
		} else if !ignoreWaitingReasons[stateWaiting.Reason] {
			state.appendMessage("%v still waiting due to %v: %v", title, stateWaiting.Reason, wrapTemporal(stateWaiting.Message))
			shouldCollectLogs = true
		}
	}

	if containerStatus.RestartCount > context.config.PodRestartGraceCount {
		runProblems = true
		stateTerminated = containerStatus.LastTerminationState.Terminated
		prefix := title
		if stateWaiting != nil {
			prefix = fmt.Sprintf("%v is in %v:", title, stateWaiting.Reason)
		}
		if stateTerminated != nil {
			state.appendMessage("%v restarted %v times, last exit due to %v (exit code %v)", prefix, wrapTemporal(containerStatus.RestartCount), stateTerminated.Reason, stateTerminated.ExitCode)
		} else {
			state.appendMessage("%v restarted %v times", prefix, wrapTemporal(containerStatus.RestartCount))
		}
		shouldCollectLogs = true
	}

	if shouldCollectLogs && context.client != nil {
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
	return
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
	} else if podPhase != v1.PodRunning && podPhase != v1.PodPending {
		state.appendMessage("Pod is in %v phase", podPhase)
	}

	state.checkContainerStatuses(pod, context)

	return
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

var eventReasonsToIgnore = map[string]bool{
	"NodeSysctlChange": true,
	"ContainerdStart":  true,
	"DockerStart":      true,
	"KubeletStart":     true,
}

func (context *diagContext) eventState(event *v1.Event) (state *eventState, err error) {
	var eName entityName
	if event.InvolvedObject.Name != "" {
		eName.namespace = event.InvolvedObject.Namespace
		eName.kind = event.InvolvedObject.Kind
		eName.name = event.InvolvedObject.Name
	} else {
		eName.kind = "Cluster"
	}
	state = context.addEventState(eName)

	if event.Type == "Normal" ||
		eventReasonsToIgnore[event.Reason] ||
		(event.Reason == "NodeNotReady" && event.Message == "Node is not ready") {
		return state, nil
	}

	source := event.Source.Component
	if source == "" {
		source = event.ReportingController
	}

	firstTimestamp := event.FirstTimestamp.Time
	if firstTimestamp.IsZero() {
		firstTimestamp = event.EventTime.Time
	}

	lastTimestamp := event.EventTime.Time
	count := int32(1)
	if event.Series != nil && !event.Series.LastObservedTime.IsZero() {
		lastTimestamp = event.Series.LastObservedTime.Time
		count = event.Series.Count
	} else if event.Count > 1 && !event.LastTimestamp.IsZero() {
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
		"since %v, %v",
		wrapTemporal(formatTime(firstTimestamp, context.config.TimeFormat, context.config.Locale)),
		wrapTemporal(formatDuration(firstTimestamp, context.now)),
	))

	if firstTimestamp != lastTimestamp && !lastTimestamp.IsZero() {
		builder.WriteString(wrapTemporal(fmt.Sprintf(
			" (last seen %v)",
			formatDuration(lastTimestamp, context.now),
		)))
	}

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
