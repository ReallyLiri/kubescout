package diag

import (
	"KubeScout/kubeclient"
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestPodState_ForEmptyPodsList(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "empty.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.Empty(t, pods)
}

func TestPodState_ForSingleCompletedPod(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "completed_single.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)
	require.Len(t, pods, 1)

	verifyAllPodsHealthy(t, pods, asTime("2021-07-18T09:00:00Z"))
}

func TestPodState_ForSingleHealthyPod(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "healthy_single.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)
	require.Len(t, pods, 1)

	verifyAllPodsHealthy(t, pods, asTime("2021-10-18T09:00:00Z"))
}

func TestPodState_ForManyHealthyPods(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "healthy_many.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)
	require.Len(t, pods, 15)

	verifyAllPodsHealthy(t, pods, asTime("2021-10-18T09:00:00Z"))
}

func TestPodState_PodPending(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "pending.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	verifyAllPodsHealthy(t, pods, asTime("2021-07-18T07:14:00Z"))
}

func TestPodState_PodPendingTooLong(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "pending.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	pendingPod := pods[0]
	state, err := testContext().podState(&pendingPod, asTime("2021-07-18T07:15:00Z"), nil)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Pod is in Pending phase", messages[0])
	require.True(t, strings.HasPrefix(messages[1], "Containers Not Ready: containers with unready status: [memory-bomb-container] (last transition: 1 minute ago)"))
}

func TestPodState_StuckInitializing(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "stuck_initializing.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-07-19T14:00:00Z")
	verifyPodsHealthyExcept(t, pods, now, map[int]bool{
		0: true,
		2: true,
		4: true,
		6: true,
		7: true,
		8: true,
	})

	pendingInitializationIndexes := []int{
		2, 4, 7, 8,
	}

	for _, index := range pendingInitializationIndexes {
		podStuckInitializing := pods[index]
		state, err := testContext().podState(&podStuckInitializing, now, nil)
		require.Nil(t, err)
		fmt.Print(state)
		require.False(t, state.IsHealthy())
		require.NotEmpty(t, state.FullName)
		messages := state.Messages()
		require.NotEmpty(t, messages)
		require.Equal(t, 3, len(messages))
		require.Equal(t, "Pod is in Pending phase", messages[0])
		require.True(t, strings.HasPrefix(messages[1], "Containers Not Initialized: containers with incomplete status: ["))
		require.True(t, strings.HasPrefix(messages[2], "Containers Not Ready: containers with unready status: ["))
	}
}

func TestPodState_EvictedOnInodes(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "evicted_inodes.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-07-16T10:00:00Z")
	verifyPodsHealthyExcept(t, pods, now, map[int]bool{
		6: true,
	})

	evictedPod := pods[6]
	state, err := testContext().podState(&evictedPod, now, nil)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "Pod is in Failed phase due to Evicted: The node was low on resource: inodes.", messages[0])
}

func TestPodState_EvictedOnMemory(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "evicted_memory.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-07-18T14:00:00Z")
	verifyPodsHealthyExcept(t, pods, now, map[int]bool{
		0: true,
	})

	evictedPod := pods[0]
	state, err := testContext().podState(&evictedPod, now, nil)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "Pod is in Failed phase due to Evicted: The node was low on resource: memory. Container memory-bomb-container was using 24GB, which exceeds its request of 0.", messages[0])
}

func TestPodState_EvictedOnDiskPressure(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "evicted_disk.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-07-18T14:00:00Z")

	verifyPodsHealthyExcept(t, pods, now, map[int]bool{
		13: true,
	})

	evictedPod := pods[13]
	state, err := testContext().podState(&evictedPod, now, nil)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "Pod is in Failed phase due to Evicted: The node was low on resource: ephemeral-storage. Container queue-consumer was using 811kB, which exceeds its request of 0. Container app6 was using 103GB, which exceeds its request of 0.", messages[0])
}

func TestPodState_EvictedOnMemory_ButContainerIsTooNew(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "evicted_memory.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-07-18T10:58:10Z") // 10 sec after pod was created

	verifyAllPodsHealthy(t, pods, now)
}

func TestPodState_CreateFailed(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "create_failed.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-08-03T08:15:00Z")

	verifyPodsHealthyExcept(t, pods, now, map[int]bool{
		6:  true,
		9:  true,
		10: true,
		11: true,
	})

	mockClient, err := kubeclient.CreateMockClient("", "", "", "", "")
	require.Nil(t, err)

	failingPod := pods[6]
	state, err := testContext().podState(&failingPod, now, mockClient)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "queue-consumer still waiting due to CreateContainerError: context deadline exceeded", messages[0])
	require.Equal(t, 1, len(state.logsCollections))
	require.Equal(t, "nxgn/app6-go-6595586ddf-5t9hx/queue-consumer/logs", state.logsCollections["queue-consumer"])
}

func TestPodState_Creating(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "creating.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-10-11T16:10:00Z")

	verifyAllPodsHealthy(t, pods, now)
}

func TestPodState_InitContainerCrashlooping(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "init_crashloop.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-10-10T00:00:00Z")

	crashloopingIndexes := []int{
		0, 1, 2, 6, 13,
	}
	initializingIndexes := []int{
		3, 4, 5, 9, 10,
	}

	skipIndexes := make(map[int]bool, len(crashloopingIndexes)+len(initializingIndexes))
	for _, index := range crashloopingIndexes {
		skipIndexes[index] = true
	}
	for _, index := range initializingIndexes {
		skipIndexes[index] = true
	}
	verifyPodsHealthyExcept(t, pods, now, skipIndexes)

	mockClient, err := kubeclient.CreateMockClient("", "", "", "", "")
	require.Nil(t, err)

	for _, index := range crashloopingIndexes {
		crashingPod := pods[index]
		state, err := testContext().podState(&crashingPod, now, mockClient)
		require.Nil(t, err)
		fmt.Printf("%v) %v", index, state)
		require.False(t, state.IsHealthy())
		require.NotEmpty(t, state.FullName)
		messages := state.Messages()
		require.NotEmpty(t, messages)
		require.Equal(t, 3, len(messages))
		require.Equal(t, "Pod is in Pending phase", messages[0])
		require.Equal(t, "wait-for-database (init) still waiting due to CrashLoopBackOff: back-off 5m0s restarting failed container", messages[1])
		require.Equal(t, "wait-for-database (init) had restarted 6 times last exit due to Error (exit code 1)", messages[2])
		require.Equal(t, 1, len(state.logsCollections))
		require.True(t, strings.HasPrefix(state.logsCollections["wait-for-database"], "gp/"))
		require.True(t, strings.HasSuffix(state.logsCollections["wait-for-database"], "/wait-for-database/logs"))
	}

	for _, index := range initializingIndexes {
		initializingPods := pods[index]
		state, err := testContext().podState(&initializingPods, now, mockClient)
		require.Nil(t, err)
		fmt.Printf("%v) %v", index, state)
		require.False(t, state.IsHealthy())
		require.NotEmpty(t, state.FullName)
		messages := state.Messages()
		require.NotEmpty(t, messages)
		if len(messages) == 2 {
			require.Equal(t, "Pod is in Pending phase", messages[0])
			require.True(t, strings.HasPrefix(messages[1], "Containers Not Ready: containers with unready status: ["))
		} else {
			require.Equal(t, 3, len(messages))
			require.Equal(t, "Pod is in Pending phase", messages[0])
			require.True(t, strings.HasPrefix(messages[1], "Containers Not Initialized: containers with incomplete status: ["))
			require.True(t, strings.HasPrefix(messages[2], "Containers Not Ready: containers with unready status: ["))
		}
		require.Equal(t, 0, len(state.logsCollections))
	}
}

func TestPodState_ExcessiveRestarts(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "excessive_restart.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-10-16T10:00:00Z")
	verifyPodsHealthyExcept(t, pods, now, map[int]bool{
		3:  true,
		4:  true,
		9:  true,
		11: true,
		13: true,
	})

	restartingPod := pods[11]
	state, err := testContext().podState(&restartingPod, now, nil)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "queue had restarted 5 times last exit due to Error (exit code 137)", messages[0])

	restartingPod = pods[13]
	state, err = testContext().podState(&restartingPod, now, nil)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages = state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "app9 had restarted 7 times last exit due to OOMKilled (exit code 137)", messages[0])
}

func TestPodState_ExcessiveRestartsForInitContainers(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "excessive_restart_init.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	unhealthyIndexes := []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 16, 18,
	}

	skipIndexes := make(map[int]bool, len(unhealthyIndexes))
	for _, index := range unhealthyIndexes {
		skipIndexes[index] = true
	}

	now := asTime("2021-07-18T07:42:00Z")
	verifyPodsHealthyExcept(t, pods, now, skipIndexes)

	restartingPod := pods[0]
	state, err := testContext().podState(&restartingPod, now, nil)
	require.Nil(t, err)
	fmt.Printf("%v) %v", 0, state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Pod is in Pending phase", messages[0])
	require.Equal(t, "run-migrations (init) had restarted 5 times", messages[1])

	pendingPod := pods[1]
	state, err = testContext().podState(&pendingPod, now, nil)
	require.Nil(t, err)
	fmt.Printf("%v) %v", 1, state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages = state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Pod is in Pending phase", messages[0])
	require.Equal(t, "Containers Not Ready: containers with unready status: [app10 queue-consumer] (last transition: 5 minutes ago)", messages[1])
}

func TestPodState_PodUnschedulableDueToInsufficientMemory(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "nodes_unavailable.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-07-18T07:42:00Z")

	unschedulablePod := pods[0]
	state, err := testContext().podState(&unschedulablePod, now, nil)
	require.Nil(t, err)
	fmt.Print(state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Pod is in Pending phase", messages[0])
	require.Equal(t, "Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 15 minutes ago)", messages[1])
}

func TestPodState_JobFailed(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "job_failed.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-08-01T00:00:00Z")

	verifyPodHealthy(t, &pods[0], now, 0)

	for i, failedPod := range pods {
		if i == 0 {
			continue
		}
		state, err := testContext().podState(&failedPod, now, nil)
		require.Nil(t, err)
		fmt.Printf("%v) %v", i, state)
		require.False(t, state.IsHealthy())
		require.NotEmpty(t, state.FullName)
		messages := state.Messages()
		require.NotEmpty(t, messages)
		require.Equal(t, 2, len(messages))
		require.Equal(t, "Pod is in Failed phase", messages[0])
		require.Equal(t, "smoke-tester terminated due to Error (exit code 1)", messages[1])
	}
}

func TestPodState_PodsStuckTerminating(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "terminating.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-10-05T16:55:00Z")

	for i, terminatingPod := range pods {
		state, err := testContext().podState(&terminatingPod, now, nil)
		require.Nil(t, err)
		fmt.Printf("%v) %v", i, state)
		require.False(t, state.IsHealthy())
		require.NotEmpty(t, state.FullName)
		messages := state.Messages()
		require.NotEmpty(t, messages)
		require.GreaterOrEqual(t, len(messages), 1)
		require.Equal(t, "Pod is Terminating since 9 minutes ago (deletion grace is 30 sec)", messages[0])
	}
}

func TestPodState_MultipleProblems(t *testing.T) {
	pods, err := kubeclient.GetPods(t, "multiple_problems.json")
	require.Nil(t, err)
	require.NotNil(t, pods)
	require.NotEmpty(t, pods)

	now := asTime("2021-10-12T12:05:00Z")

	state, err := testContext().podState(&pods[0], now, nil)
	require.Nil(t, err)
	fmt.Printf("%v) %v", 0, state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages := state.Messages()
	require.NotEmpty(t, messages)
	require.GreaterOrEqual(t, len(messages), 2)
	require.Equal(t, "Pod is in Pending phase", messages[0])
	require.Equal(t, "nginx-1 still waiting due to ImagePullBackOff: Back-off pulling image \"nginx:l4t3st\"", messages[1])

	state, err = testContext().podState(&pods[1], now, nil)
	require.Nil(t, err)
	fmt.Printf("%v) %v", 1, state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages = state.Messages()
	require.NotEmpty(t, messages)
	require.GreaterOrEqual(t, len(messages), 2)
	require.Equal(t, "Pod is in Pending phase", messages[0])
	require.Equal(t, "Unschedulable: 0/7 nodes are available: 7 Insufficient memory. (last transition: 4 minutes ago)", messages[1])

	state, err = testContext().podState(&pods[2], now, nil)
	require.Nil(t, err)
	fmt.Printf("%v) %v", 2, state)
	require.False(t, state.IsHealthy())
	require.NotEmpty(t, state.FullName)
	messages = state.Messages()
	require.NotEmpty(t, messages)
	require.GreaterOrEqual(t, len(messages), 2)
	require.Equal(t, "Pod is in Pending phase", messages[0])
	require.Equal(t, "Containers Not Ready: containers with unready status: [nginx-3] (last transition: 4 minutes ago)", messages[1])
}
