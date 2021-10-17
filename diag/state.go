package diag

import (
	"KubeScout/config"
	"KubeScout/kubeclient"
	"fmt"
	"github.com/goombaio/orderedset"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"strconv"
	"strings"
	"time"
)

func (state *EntityState) checkStatuses(pod *v1.Pod, statuses []v1.ContainerStatus, initContainers bool, config *config.Config) bool {
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
			state.appendMessage("%v%v still waiting due to %v: %v", containerStatus.Name, subTitle, stateWaiting.Reason, message)
			anyMessageForContainer = true
		}
		if containerStatus.RestartCount > config.PodRestartGraceCount {
			stateTerminated = containerStatus.LastTerminationState.Terminated
			if stateTerminated != nil {
				state.appendMessage("%v%v had restarted %v times last exit due to %v (exit code %v)", containerStatus.Name, subTitle, containerStatus.RestartCount, stateTerminated.Reason, stateTerminated.ExitCode)
			} else {
				state.appendMessage("%v%v had restarted %v times", containerStatus.Name, subTitle, containerStatus.RestartCount)
			}
			anyMessageForContainer = true
		}
		if anyMessageForContainer && state.client != nil {
			logs, err := state.client.GetPodLogs(pod.Namespace, pod.Name, containerStatus.Name)
			if err != nil {
				fmt.Printf("failed to get logs of %v.%v.%v: %v", pod.Namespace, pod.Name, containerStatus.Name, err)
			} else if logs != "" {
				state.logsCollections[containerStatus.Name] = logs
			}
		}
		anyMessage = anyMessage || anyMessageForContainer
	}
	return anyMessage
}

func (context *diagContext) podState(pod *v1.Pod, now time.Time, client kubeclient.KubernetesClient) (state *EntityState, err error) {
	state = &EntityState{
		FullName:        fmt.Sprintf("%v/%v", pod.Namespace, pod.Name),
		Kind:            "Pod",
		messages:        orderedset.NewOrderedSet(),
		ActionTaken:     false,
		client:          client,
		logsCollections: map[string]string{},
	}

	podPhase := pod.Status.Phase
	if podPhase == v1.PodSucceeded {
		return
	}

	baselineTime := pod.ObjectMeta.CreationTimestamp.Time
	if pod.Status.StartTime != nil {
		baselineTime = pod.Status.StartTime.Time
	}
	sinceCreation := now.Sub(baselineTime).Seconds()
	if sinceCreation < context.config.PodCreationGracePeriodSeconds {
		return
	}

	statusReason := pod.Status.Reason
	if statusReason != "" {
		statusMessage := pod.Status.Message
		if statusReason == "Evicted" {
			statusMessage = formatUnitsSize(statusMessage)
		}
		state.appendMessage("Pod is in %v phase due to %v: %v", podPhase, statusReason, statusMessage)
	} else if pod.DeletionTimestamp != nil {
		deletionTime := (*pod.DeletionTimestamp).Time
		if now.Sub(deletionTime).Seconds() > float64(valueOrDefault(pod.DeletionGracePeriodSeconds, context.config.PodTerminationGracePeriodSeconds)) {
			suffix := ""
			if pod.DeletionGracePeriodSeconds != nil && *pod.DeletionGracePeriodSeconds != 0 {
				suffix = fmt.Sprintf(" (deletion grace is %v sec)", *pod.DeletionGracePeriodSeconds)
			}
			state.appendMessage("Pod is Terminating since %v%v", formatDuration(deletionTime, now), suffix)
		}
	} else if podPhase != v1.PodRunning {
		state.appendMessage("Pod is in %v phase", podPhase)
	}

	anyStatusMessage := state.checkStatuses(pod, pod.Status.ContainerStatuses, false, context.config)
	anyStatusMessage = anyStatusMessage || state.checkStatuses(pod, pod.Status.InitContainerStatuses, true, context.config)

	if !anyStatusMessage && pod.DeletionTimestamp == nil {
		for _, condition := range pod.Status.Conditions {
			if condition.Status != "True" {
				state.appendMessage("%v: %v (last transition: %v)", splitToWords(condition.Reason), condition.Message, formatDuration(condition.LastTransitionTime.Time, now))
			}
		}
	}

	return
}

func (context *diagContext) eventState(event *v1.Event, now time.Time) (state *EntityState, err error) {
	state = &EntityState{
		FullName:        fmt.Sprintf("%v/%v", event.Namespace, event.Name),
		Kind:            "Event",
		messages:        orderedset.NewOrderedSet(),
		ActionTaken:     false,
		logsCollections: map[string]string{},
	}

	if event.Type == "Normal" {
		return state, nil
	}

	involvedObject := event.InvolvedObject
	var suffix string
	var eventMessageLines []string
	if event.Message != "" {
		eventMessageLines = strings.Split(event.Message, "\n")
		suffix = ":"
	}

	state.appendMessage(
		"Event on %v %v due to %v (at %v, %v)%v",
		involvedObject.Kind,
		involvedObject.Name,
		event.Reason,
		formatTime(event.LastTimestamp.Time, context.config.TimeFormat),
		formatDuration(event.LastTimestamp.Time, now),
		suffix,
	)
	for _, messageLine := range eventMessageLines {
		state.appendMessage("\t%v", messageLine)
	}
	return state, nil
}

func (context *diagContext) nodeState(node *v1.Node, now time.Time, forceCheckResources bool) (state *EntityState, err error) {
	state = &EntityState{
		FullName:        node.Name,
		Kind:            "Node",
		messages:        orderedset.NewOrderedSet(),
		ActionTaken:     false,
		logsCollections: map[string]string{},
	}

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
		state.appendMessage("%v: %v (last transition: %v)", splitToWords(condition.Reason), formatUnitsSize(condition.Message), formatDuration(condition.LastTransitionTime.Time, now))
	}

	if !state.IsHealthy() && !forceCheckResources {
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

func (context *diagContext) replicaSetState(replicaSet *v12.ReplicaSet, now time.Time) (state *EntityState, err error) {
	state = &EntityState{
		FullName:        fmt.Sprintf("%v/%v", replicaSet.Namespace, replicaSet.Name),
		Kind:            "ReplicaSet",
		messages:        orderedset.NewOrderedSet(),
		ActionTaken:     false,
		logsCollections: map[string]string{},
	}

	specDesiredReplicas := replicaSet.Spec.Replicas
	var desiredReplicas int
	if specDesiredReplicas == nil {
		desiredReplicasAnnotation, found := replicaSet.ObjectMeta.Annotations["deployment.kubernetes.io/desired-replicas"]
		if found {
			desiredReplicas, err = strconv.Atoi(desiredReplicasAnnotation)
			if err != nil {
				fmt.Printf("Failed to parse desired replicas annotation value '%v': %v", desiredReplicasAnnotation, err)
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
		state.appendMessage("%v: %v (last transition: %v)", splitToWords(condition.Reason), formatUnitsSize(condition.Message), formatDuration(condition.LastTransitionTime.Time, now))
	}
	return
}
