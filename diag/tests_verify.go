package diag

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
	"time"
)

func verifyEventsHealthyExcept(t *testing.T, events []v1.Event, now time.Time, unhealthyIndexes map[int]bool) {
	for i, event := range events {
		if _, skip := unhealthyIndexes[i]; skip {
			continue
		}
		verifyEventHealthy(t, &event, now, i)
	}
}

func verifyEventHealthy(t *testing.T, event *v1.Event, now time.Time, index int) {
	state, err := testContext(now).eventState(event)
	assert.Nil(t, err)
	log.Debugf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.name.Kind)
	assert.Empty(t, state.message)
}

func verifyAllNodesHealthy(t *testing.T, nodes []v1.Node, now time.Time) {
	verifyNodesHealthyExcept(t, nodes, now, map[int]bool{})
}

func verifyNodesHealthyExcept(t *testing.T, nodes []v1.Node, now time.Time, unhealthyIndexes map[int]bool) {
	for i, node := range nodes {
		if _, skip := unhealthyIndexes[i]; skip {
			continue
		}
		verifyNodeHealthy(t, node, now, i)
	}
}

func verifyNodeHealthy(t *testing.T, node v1.Node, now time.Time, index int) {
	state, err := testContext(now).nodeState(&node, true)
	assert.Nil(t, err)
	log.Debugf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.name)
	assert.Empty(t, state.cleanMessages())
}

func verifyAllPodsHealthy(t *testing.T, pods []v1.Pod, now time.Time) {
	verifyPodsHealthyExcept(t, pods, now, map[int]bool{})
}

func verifyPodsHealthyExcept(t *testing.T, pods []v1.Pod, now time.Time, unhealthyIndexes map[int]bool) {
	for i, pod := range pods {
		if _, skip := unhealthyIndexes[i]; skip {
			continue
		}
		verifyPodHealthy(t, &pod, now, i)
	}
}

func verifyPodHealthy(t *testing.T, pod *v1.Pod, now time.Time, index int) {
	state, err := testContext(now).podState(pod)
	assert.Nil(t, err)
	log.Debugf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.name)
	assert.Empty(t, state.cleanMessages())
}

func verifyAllReplicaSetsHealthy(t *testing.T, replicaSets []v12.ReplicaSet, now time.Time) {
	verifyReplicaSetsHealthyExcept(t, replicaSets, now, map[int]bool{})
}

func verifyReplicaSetsHealthyExcept(t *testing.T, replicaSets []v12.ReplicaSet, now time.Time, unhealthyIndexes map[int]bool) {
	for i, replicaSet := range replicaSets {
		if _, skip := unhealthyIndexes[i]; skip {
			continue
		}
		verifyReplicaSetHealthy(t, replicaSet, now, i)
	}
}

func verifyReplicaSetHealthy(t *testing.T, replicaSet v12.ReplicaSet, now time.Time, index int) {
	state, err := testContext(now).replicaSetState(&replicaSet)
	assert.Nil(t, err)
	log.Debugf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.name)
	assert.Empty(t, state.cleanMessages())
}
