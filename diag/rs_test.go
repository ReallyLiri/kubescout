package diag

import (
	"github.com/reallyliri/kubescout/kubeclient"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestReplicaSetState_AllHealthy(t *testing.T) {
	sets, err := kubeclient.GetReplicaSets(t, "healthy.json")
	require.Nil(t, err)
	require.NotNil(t, sets)
	require.NotEmpty(t, sets)
	require.Equal(t, 238, len(sets))

	verifyAllReplicaSetsHealthy(t, sets, asTime("2021-10-11T12:50:00Z"))
}

func TestReplicaSetState_ExceededQuota(t *testing.T) {
	sets, err := kubeclient.GetReplicaSets(t, "quota_exceeded.json")
	require.Nil(t, err)
	require.NotNil(t, sets)
	require.NotEmpty(t, sets)
	require.Equal(t, 19, len(sets))

	errorSets := []int{
		0, 1, 2, 3, 5, 6, 10, 13,
	}

	skipIndexes := make(map[int]bool, len(errorSets))
	for _, index := range errorSets {
		skipIndexes[index] = true
	}

	now := asTime("2021-07-27T11:35:00Z")

	verifyReplicaSetsHealthyExcept(t, sets, now, skipIndexes)

	for _, index := range errorSets {
		state, err := testContext().replicaSetState(&sets[index], now)
		require.Nil(t, err)
		log.Debugf("%v) %v", index, state)
		require.False(t, state.isHealthy())
		require.NotEmpty(t, state.fullName)
		messages := state.messages
		require.NotEmpty(t, messages)
		require.Equal(t, 1, len(messages))
		require.True(t, strings.HasPrefix(messages[0], "Failed Create: pods \""))
		require.True(t, strings.HasSuffix(messages[0], " minutes ago)"))

		switch index {
		case 0:
			require.True(t, strings.Index(messages[0], "is forbidden: exceeded quota: resource-quota, requested: memory=1.0MB, used: memory=13GB, limited: memory=14MB (last transition: ") > 0)
		case 1:
		case 3:
			require.True(t, strings.Index(messages[0], "is forbidden: exceeded quota: resource-quota, requested: cpu=0.6,memory=1.3GB, used: cpu=6.5,memory=13GB, limited: cpu=7,memory=14MB (last transition: ") > 0)
		case 2:
			require.True(t, strings.Index(messages[0], "is forbidden: exceeded quota: resource-quota, requested: cpu=0.8,memory=1.6GB, used: cpu=6.3,memory=13GB, limited: cpu=7,memory=14MB (last transition: ") > 0)
		case 5:
			require.True(t, strings.Index(messages[0], "is forbidden: exceeded quota: resource-quota, requested: cpu=1.1,memory=2.4GB, used: cpu=6.5,memory=13GB, limited: cpu=7,memory=14MB (last transition: ") > 0)
		case 6:
			require.True(t, strings.Index(messages[0], "is forbidden: exceeded quota: resource-quota, requested: cpu=1.1,memory=2.4GB, used: cpu=6.3,memory=13GB, limited: cpu=7,memory=14MB (last transition: ") > 0)
		case 10:
			require.True(t, strings.Index(messages[0], "is forbidden: exceeded quota: resource-quota, requested: cpu=0.8,memory=1.6GB, used: cpu=6.5,memory=13GB, limited: cpu=7,memory=14MB (last transition: ") > 0)
		case 13:
			require.True(t, strings.Index(messages[0], "is forbidden: exceeded quota: resource-quota, requested: cpu=0.5, used: cpu=6.5, limited: cpu=7 (last transition: ") > 0)

		}
	}
}
