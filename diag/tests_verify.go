package diag

import (
	"github.com/stretchr/testify/assert"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"log"
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
	state, err := testContext().eventState(event, now)
	assert.Nil(t, err)
	log.Printf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.fullName)
	assert.Empty(t, state.messages)
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
	state, err := testContext().nodeState(&node, now, true)
	assert.Nil(t, err)
	log.Printf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.fullName)
	assert.Empty(t, state.messages)
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
	state, err := testContext().podState(pod, now, nil)
	assert.Nil(t, err)
	log.Printf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.fullName)
	assert.Empty(t, state.messages)
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
	state, err := testContext().replicaSetState(&replicaSet, now)
	assert.Nil(t, err)
	log.Printf("%v) %v", index, state)
	assert.True(t, state.isHealthy())
	assert.NotEmpty(t, state.fullName)
	assert.Empty(t, state.messages)
}
