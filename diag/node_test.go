package diag

import (
	"github.com/reallyliri/kubescout/kubeclient"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNodeState_AllHealthy(t *testing.T) {
	nodes, err := kubeclient.GetNodes(t, "healthy.json")
	require.Nil(t, err)
	require.NotNil(t, nodes)
	require.NotEmpty(t, nodes)
	require.Equal(t, 3, len(nodes))

	verifyAllNodesHealthy(t, nodes, asTime("2021-10-11T12:50:00Z"))
}

func TestNodeState_NodeInUnknownState(t *testing.T) {
	nodes, err := kubeclient.GetNodes(t, "unknown.json")
	require.Nil(t, err)
	require.NotNil(t, nodes)
	require.NotEmpty(t, nodes)

	now := asTime("2021-10-13T15:00:00Z")
	state, err := testContext().nodeState(&nodes[0], now, true)
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name)
	messages := state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "Node Status Unknown: Kubelet stopped posting node status. (last transition: 55 minutes ago)", messages[0])
}

func TestNodeState_ExcessiveCpu(t *testing.T) {
	nodes, err := kubeclient.GetNodes(t, "excessive_memory.json")
	require.Nil(t, err)
	require.NotNil(t, nodes)
	require.NotEmpty(t, nodes)

	now := asTime("2021-07-19T15:00:00Z")

	state, err := testContext().nodeState(&nodes[0], now, true)
	require.Nil(t, err)
	log.Debug(state.String())
	require.False(t, state.isHealthy())
	require.NotEmpty(t, state.name)
	messages := state.messages
	require.NotEmpty(t, messages)
	require.Equal(t, 1, len(messages))
	require.Equal(t, "Excessive usage of Memory: 54GB/55GB (99.2% usage)", messages[0])
}
