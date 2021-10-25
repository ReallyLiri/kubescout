package diag

import (
	"github.com/reallyliri/kubescout/internal"
	"github.com/reallyliri/kubescout/kubeclient"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestEventState_StandardEvents(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "standard.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 161, len(events))

	now := asTime("2021-10-12T13:55:00Z")

	verifyEventsHealthyExcept(t, events, now, map[int]bool{
		139: true,
	})

	state, err := testContext().eventState(&events[139], now)
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages := strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 4, len(messages))
	require.Equal(t, "Event by kubelet: Unhealthy x2 since 12 Oct 21 13:54 UTC (last seen 26 seconds ago):", messages[0])
	require.Equal(t, "\tLiveness probe failed:   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current", messages[1])
	require.Equal(t, "\tDload  Upload   Total   Spent    Left  Speed", messages[2])
	require.Equal(t, "\t0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0curl: (7) Failed to connect to localhost port 8095: Connection refused", messages[3])
}

func TestEventState_MountFailedEvents(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "mount_failed.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 13, len(events))

	now := asTime("2021-10-12T13:30:00Z")

	warningIndexes := []int{
		1, 10, 11, 12,
	}
	skipIndexes := internal.ToMap(warningIndexes)

	verifyEventsHealthyExcept(t, events, now, skipIndexes)

	state, err := testContext().eventState(&events[1], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 1, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages := strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by kubelet: Failed x351 since 12 Oct 21 12:00 UTC (last seen 9 minutes ago):", messages[0])
	require.Equal(t, "\tError: ImagePullBackOff", messages[1])

	state, err = testContext().eventState(&events[10], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 10, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages = strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by default-scheduler: FailedScheduling x476 since 12 Oct 21 12:01 UTC (last seen 4 minutes ago):", messages[0])
	require.Equal(t, "\t0/7 nodes are available: 7 Insufficient memory.", messages[1])

	state, err = testContext().eventState(&events[11], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 11, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages = strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by kubelet: FailedMount x10 since 12 Oct 21 12:02 UTC (last seen 3 minutes ago):", messages[0])
	require.Equal(t, "\tUnable to attach or mount volumes: unmounted volumes=[nginx-pvc], unattached volumes=[default-token-6xwwv nginx-pvc]: timed out waiting for the condition", messages[1])

	state, err = testContext().eventState(&events[12], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 12, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages = strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by kubelet: FailedMount x28 since 12 Oct 21 12:05 UTC (last seen 5 minutes ago):", messages[0])
	require.Equal(t, "\tUnable to attach or mount volumes: unmounted volumes=[nginx-pvc], unattached volumes=[nginx-pvc default-token-6xwwv]: timed out waiting for the condition", messages[1])
}

func TestEventState_NodeProblemDetector(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "npd.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 4, len(events))

	now := asTime("2021-10-14T06:30:00Z")

	state, err := testContext().eventState(&events[0], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 0, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages := strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 1, len(messages))
	require.Equal(t, "Event by sysctl-monitor: NodeSysctlChange x29 since 07 Oct 21 05:24 UTC (last seen 1 hour ago)", messages[0])

	state, err = testContext().eventState(&events[1], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 1, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages = strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by kernel-monitor: KernelOops since 14 Oct 21 06:10 UTC (last seen 19 minutes ago):", messages[0])
	require.Equal(t, "\tkernel: BUG: unable to handle kernel NULL pointer dereference at TESTING", messages[1])
}

func TestEventState_FailedJobs(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "failed_jobs.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 75, len(events))

	now := asTime("2021-10-21T11:00:00Z")

	verifyEventsHealthyExcept(t, events, now, map[int]bool{
		71: true,
	})

	state, err := testContext().eventState(&events[71], now)
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.involvedObject)
	messages := strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by job-controller: BackoffLimitExceeded since 21 Oct 21 10:06 UTC (last seen 53 minutes ago):", messages[0])
	require.Equal(t, "\tJob has reached the specified backoff limit", messages[1])
}

func TestEventState_LivenessFailed(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "liveness_failed.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 134, len(events))

	now := asTime("2021-10-19T09:00:00Z")

	warningIndexes := []int{
		8,
		18,
		29,
		30,
		31,
		32,
		33,
		36,
		47,
		48,
		49,
		52,
		59,
		67,
		68,
		69,
		82,
		109,
		112,
		113,
		114,
		124,
		131,
		132,
	}

	skipIndexes := internal.ToMap(warningIndexes)

	verifyEventsHealthyExcept(t, events, now, skipIndexes)

	for _, i := range warningIndexes {
		state, err := testContext().eventState(&events[i], now)
		require.Nil(t, err)
		log.Debug(state.String())
		require.False(t, state.isHealthy())
		require.NotEmpty(t, state.involvedObject)
		messages := strings.Split(state.cleanMessage(), "\n")
		require.True(t, len(messages) >= 2)
		require.True(t, len(messages) <= 5)
		require.True(t, strings.HasPrefix(messages[0], "Event by kubelet: "))
		require.True(t, strings.Index(messages[0], "Unhealthy ") > 0 || strings.Index(messages[0], "BackOff ") > 0)
		require.True(t,
			strings.HasPrefix(messages[1], "\tLiveness probe errored:") ||
				strings.HasPrefix(messages[1], "\tLiveness probe failed:") ||
				strings.HasPrefix(messages[1], "\tBack-off restarting failed container"),
		)
	}
}
