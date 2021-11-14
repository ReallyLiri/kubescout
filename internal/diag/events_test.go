package diag

import (
	"github.com/reallyliri/kubescout/internal"
	"github.com/reallyliri/kubescout/internal/kubeclient"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

	state, err := testContext(now).eventState(&events[139])
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages := strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 4, len(messages))
	require.Equal(t, "Event by kubelet: Unhealthy x2 since 12 Oct 21 13:54 UTC, 41 seconds ago (last seen 26 seconds ago):", messages[0])
	require.Equal(t, "\tLiveness probe failed:   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current", messages[1])
	require.Equal(t, "\tDload  Upload   Total   Spent    Left  Speed", messages[2])
	require.Equal(t, "\t0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0curl: (7) Failed to connect to localhost port 8095: Connection refused", messages[3])
}

func TestEventState_MountFailedTimeout(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "mount_failed_timeout.json")
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

	state, err := testContext(now).eventState(&events[1])
	require.Nil(t, err)
	log.Debugf("%v) %v", 1, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages := strings.Split(state.cleanMessage(), "\n")
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, "Event by kubelet: Failed x351 since 12 Oct 21 12:00 UTC, 1 hour ago (last seen 9 minutes ago):", messages[0])
	assert.Equal(t, "\tError: ImagePullBackOff", messages[1])

	state, err = testContext(now).eventState(&events[10])
	require.Nil(t, err)
	log.Debugf("%v) %v", 10, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages = strings.Split(state.cleanMessage(), "\n")
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, "Event by default-scheduler: FailedScheduling x476 since 12 Oct 21 12:01 UTC, 1 hour ago (last seen 4 minutes ago):", messages[0])
	assert.Equal(t, "\t0/7 nodes are available: 7 Insufficient memory.", messages[1])

	state, err = testContext(now).eventState(&events[11])
	require.Nil(t, err)
	log.Debugf("%v) %v", 11, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages = strings.Split(state.cleanMessage(), "\n")
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, "Event by kubelet: FailedMount x10 since 12 Oct 21 12:02 UTC, 1 hour ago (last seen 3 minutes ago):", messages[0])
	assert.Equal(t, "\tUnable to attach or mount volumes: unmounted volumes=[nginx-pvc], unattached volumes=[default-token-6xwwv nginx-pvc]: timed out waiting for the condition", messages[1])

	state, err = testContext(now).eventState(&events[12])
	require.Nil(t, err)
	log.Debugf("%v) %v", 12, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages = strings.Split(state.cleanMessage(), "\n")
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, "Event by kubelet: FailedMount x28 since 12 Oct 21 12:05 UTC, 1 hour ago (last seen 5 minutes ago):", messages[0])
	assert.Equal(t, "\tUnable to attach or mount volumes: unmounted volumes=[nginx-pvc], unattached volumes=[nginx-pvc default-token-6xwwv]: timed out waiting for the condition", messages[1])
}

func TestEventState_NodeProblemDetector(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "npd.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 4, len(events))

	now := asTime("2021-10-14T06:30:00Z")

	state, err := testContext(now).eventState(&events[1])
	require.Nil(t, err)
	log.Debugf("%v) %v", 1, state)
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages := strings.Split(state.cleanMessage(), "\n")
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, "Event by kernel-monitor: KernelOops since 14 Oct 21 06:10 UTC, 19 minutes ago:", messages[0])
	assert.Equal(t, "\tkernel: BUG: unable to handle kernel NULL pointer dereference at TESTING", messages[1])
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

	state, err := testContext(now).eventState(&events[71])
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages := strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by job-controller: BackoffLimitExceeded since 21 Oct 21 10:06 UTC, 53 minutes ago:", messages[0])
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
		state, err := testContext(now).eventState(&events[i])
		require.Nil(t, err)
		log.Debug(state.String())
		require.False(t, state.isHealthy())
		require.NotEmpty(t, state.name.Name)
		messages := strings.Split(state.cleanMessage(), "\n")
		require.True(t, len(messages) >= 2, len(messages))
		require.True(t, len(messages) <= 9, len(messages))
		require.True(t, strings.HasPrefix(messages[0], "Event by kubelet: "))
		require.True(t, strings.Index(messages[0], "Unhealthy ") > 0 || strings.Index(messages[0], "BackOff ") > 0)
		require.True(t,
			strings.HasPrefix(messages[1], "\tLiveness probe errored:") ||
				strings.HasPrefix(messages[1], "\tLiveness probe failed:") ||
				strings.HasPrefix(messages[1], "\tBack-off restarting failed container"),
		)
	}
}

func TestEventState_EndpointsWarning(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "endpoints.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 26, len(events))

	now := asTime("2021-10-31T08:45:30Z")

	verifyEventsHealthyExcept(t, events, now, map[int]bool{})
}

func TestEventState_StartedEventsShouldBeIgnored(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "started_events.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	require.Equal(t, 103, len(events))

	now := asTime("2021-10-31T08:45:30Z")

	verifyEventsHealthyExcept(t, events, now, map[int]bool{})
}

func TestEventState_MissingDefinitions(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "missing_definitions.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, 26, len(events))

	now := asTime("2021-11-14T11:45:00Z")

	verifyEventsHealthyExcept(t, events, now, map[int]bool{3: true, 16: true})

	state, err := testContext(now).eventState(&events[3])
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages := strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by kubelet: Failed x8 since 14 Nov 21 11:28 UTC, 16 minutes ago (last seen 15 minutes ago):", messages[0])
	require.Equal(t, "\tError: configmap \"confmap\" not found", messages[1])

	state, err = testContext(now).eventState(&events[16])
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	messages = strings.Split(state.cleanMessage(), "\n")
	require.Equal(t, 2, len(messages))
	require.Equal(t, "Event by kubelet: Failed x8 since 14 Nov 21 11:28 UTC, 16 minutes ago (last seen 15 minutes ago):", messages[0])
	require.Equal(t, "\tError: secret \"db\" not found", messages[1])
}

func TestEventState_MountFailedConnection(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "mount_failed_connection.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, 4, len(events))

	now := asTime("2021-11-01T11:45:00Z")

	verifyEventsHealthyExcept(t, events, now, map[int]bool{1: true, 2: true, 3: true})

	state, err := testContext(now).eventState(&events[2])
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	require.Equal(t, state.cleanMessage(), `Event by kubelet: FailedMount x884 since 30 Oct 21 13:45 UTC, 1 day ago (last seen 3 hours ago):
	(combined from similar events): MountVolume.SetUp failed for volume "shared-pv-7fd0ebe5431642bb9b3e7f0577" : mount failed: exit status 1
	Mounting command: systemd-run
	Mounting arguments: --description=Kubernetes transient mount for /var/lib/kubelet/pods/28755586-fab2-43a4-91b8-03ccba76a3d6/volumes/kubernetes.io~nfs/shared-pv-7fd0ebe5431642bb9b3e7f0577 --scope -- /home/kubernetes/containerized_mounter/mounter mount -t nfs 10.100.6.29:/ /var/lib/kubelet/pods/28755586-fab2-43a4-91b8-03ccba76a3d6/volumes/kubernetes.io~nfs/shared-pv-7fd0ebe5431642bb9b3e7f0577
	Output: Running scope as unit: run-r219d468d2c644f8bad1a6d7631ae4668.scope
	Mount failed: mount failed: exit status 32
	Mounting command: chroot
	Mounting arguments: [/home/kubernetes/containerized_mounter/rootfs mount -t nfs 10.100.6.29:/ /var/lib/kubelet/pods/28755586-fab2-43a4-91b8-03ccba76a3d6/volumes/kubernetes.io~nfs/shared-pv-7fd0ebe5431642bb9b3e7f0577]
	Output: mount.nfs: Connection timed out`)
}

func TestEventState_SearchLimit(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "search_limit.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, 298, len(events))

	now := asTime("2021-11-01T11:45:00Z")

	for i, event := range events {
		state, err := testContext(now).eventState(&event)
		require.Nil(t, err)
		log.Debugf("%v) %v", i, state)
		if state.isHealthy() || i == 136 {
			continue
		}

		require.NotEmpty(t, state.name.Name)
		messages := strings.Split(state.cleanMessage(), "\n")
		require.Equal(t, 2, len(messages))
		require.Equal(t, "\tSearch Line limits were exceeded, some search paths have been omitted, the applied search line is: default.svc.cluster.local svc.cluster.local cluster.local acme.int corp.acme.test acme.net", messages[1])
	}
}

func TestEventState_BrokenStartCommand(t *testing.T) {
	events, err := kubeclient.GetEvents(t, "start_command.json")
	require.Nil(t, err)
	require.NotNil(t, events)
	require.NotEmpty(t, events)
	assert.Equal(t, 10, len(events))

	now := asTime("2021-11-01T11:45:00Z")

	verifyEventsHealthyExcept(t, events, now, map[int]bool{4: true, 6: true})

	state, err := testContext(now).eventState(&events[4])
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name.Name)
	require.Equal(t, state.cleanMessage(), `Event by kubelet: Failed x3 since 14 Nov 21 11:32 UTC, now (last seen now):
	Error: failed to start container "command-broken": Error response from daemon: OCI runtime create failed: container_linux.go:380: starting container process caused: exec: "shab": executable file not found in $PATH: unknown`)
}
