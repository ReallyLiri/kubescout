package diag

import (
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
	require.NotEmpty(t, state.fullName)
	messages := state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	messages = strings.Split(messages[0], "\n")
	require.Equal(t, 4, len(messages))
	require.Equal(t, "Event on Pod app9-5965b85fc7-nchvk due to Unhealthy (at 12 Oct 21 13:54 UTC, 26 seconds ago):", messages[0])
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
	skipIndexes := make(map[int]bool, len(warningIndexes))
	for _, index := range warningIndexes {
		skipIndexes[index] = true
	}

	verifyEventsHealthyExcept(t, events, now, skipIndexes)

	state, err := testContext().eventState(&events[1], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 1, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.fullName)
	messages := state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	messages = strings.Split(messages[0], "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event on Pod nginx-1-7dfc456b4f-78mtz due to Failed (at 12 Oct 21 13:20 UTC, 9 minutes ago):", messages[0])
	require.Equal(t, "\tError: ImagePullBackOff", messages[1])

	state, err = testContext().eventState(&events[10], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 10, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.fullName)
	messages = state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	messages = strings.Split(messages[0], "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event on Pod nginx-2-6f8f94f55c-fmcjs due to FailedScheduling (at 12 Oct 21 13:25 UTC, 4 minutes ago):", messages[0])
	require.Equal(t, "\t0/7 nodes are available: 7 Insufficient memory.", messages[1])

	state, err = testContext().eventState(&events[11], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 11, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.fullName)
	messages = state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	messages = strings.Split(messages[0], "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event on Pod nginx-3-d75464d75-llslq due to FailedMount (at 12 Oct 21 13:26 UTC, 3 minutes ago):", messages[0])
	require.Equal(t, "\tUnable to attach or mount volumes: unmounted volumes=[nginx-pvc], unattached volumes=[default-token-6xwwv nginx-pvc]: timed out waiting for the condition", messages[1])

	state, err = testContext().eventState(&events[12], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 12, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.fullName)
	messages = state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	messages = strings.Split(messages[0], "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event on Pod nginx-3-d75464d75-llslq due to FailedMount (at 12 Oct 21 13:24 UTC, 5 minutes ago):", messages[0])
	require.Equal(t, "\tUnable to attach or mount volumes: unmounted volumes=[nginx-pvc], unattached volumes=[nginx-pvc default-token-6xwwv]: timed out waiting for the condition", messages[1])
}

func TestEventState_NodeProblemDetector(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "npd.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 4, len(events))

	now := asTime("2021-10-14T05:30:00Z")

	state, err := testContext().eventState(&events[0], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 0, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.fullName)
	messages := state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "Event on Node node-pool--19cbb605-22h0 due to NodeSysctlChange (at 14 Oct 21 05:24 UTC, 5 minutes ago)", messages[0])

	state, err = testContext().eventState(&events[1], now)
	require.Nil(t, err)
	log.Debugf("%v) %v", 1, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.fullName)
	messages = state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	messages = strings.Split(messages[0], "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event on Node node-pool--19cbb605-22h0 due to KernelOops (at 14 Oct 21 06:10 UTC, 40 minutes ):", messages[0])
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
	require.NotEmpty(t, state.fullName)
	messages := state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	messages = strings.Split(messages[0], "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event on Job tester-1634810400 due to BackoffLimitExceeded (at 21 Oct 21 10:06 UTC, 53 minutes ago):", messages[0])
	require.Equal(t, "\tJob has reached the specified backoff limit", messages[1])
}


func TestEventState_RpcError(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "liveness_failed.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 134, len(events))

	now := asTime("2021-10-19T09:00:00Z")

	warningIndexes := []int {

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
		133,
	}

	skipIndexes := make(map[int]bool, len(warningIndexes))
	for _, index := range warningIndexes {
		skipIndexes[index] = true
	}

	verifyEventsHealthyExcept(t, events, now, skipIndexes)

	for _, i := range warningIndexes {
		state, err := testContext().eventState(&events[i], now)
		require.Nil(t, err)
		log.Debug(state.String())
		require.False(t, state.isHealthy())
		require.NotEmpty(t, state.fullName)
		messages := state.messages
		require.NotEmpty(t, messages)
		require.Equal(t, 1, len(messages))
		messages = strings.Split(messages[0], "\n")
		require.True(t, len(messages) >= 2)
		require.True(t, len(messages) <= 5)
		require.True(t, strings.HasPrefix(messages[0], "Event on Pod"))
		require.True(t, strings.Index(messages[0], "due to Unhealthy") > 0)
		require.True(t,
			strings.HasPrefix(messages[1], "\tLiveness probe errored: ") || strings.HasPrefix(messages[1], "\tLiveness probe failed: "),
		)
	}
}
