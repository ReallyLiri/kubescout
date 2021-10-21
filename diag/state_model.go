package diag

import (
	"fmt"
	"github.com/goombaio/orderedset"
	"github.com/reallyliri/kubescout/kubeclient"
	"strings"
	"time"
)

type entityState struct {
	fullName        string
	correlationName string
	kind            string
	hashes          *orderedset.OrderedSet
	messages        []string
	logsCollections map[string]string
	client          kubeclient.KubernetesClient
	timestamp       time.Time
}

func newState(fullName string, correlationName, kind string, client kubeclient.KubernetesClient) *entityState {
	return &entityState{
		fullName:        fullName,
		correlationName: correlationName,
		kind:            kind,
		hashes:          orderedset.NewOrderedSet(),
		messages:        []string{},
		client:          client,
		logsCollections: map[string]string{},
	}
}

func (state entityState) isHealthy() bool {
	return len(state.messages) == 0
}

func (state *entityState) String() string {
	if state.isHealthy() {
		return fmt.Sprintf("%v is healthy\n", state.fullName)
	}
	messages := state.messages
	if state.kind != "Event" {
		messages = append([]string{fmt.Sprintf("%v %v is un-healthy", state.kind, state.fullName)}, messages...)
	}
	return strings.Join(messages, "\n\t")
}

func (state *entityState) appendMessage(format string, a ...interface{}) {
	var message string
	if len(a) > 0 {
		message = strings.TrimSpace(fmt.Sprintf(format, a...))
	} else {
		message = format
	}
	if message == "" {
		return
	}
	messageHash := hash(state.correlationName, normalizeMessage(message))
	if state.hashes.Contains(messageHash) {
		return
	}
	state.hashes.Add(messageHash)
	state.messages = append(state.messages, cleanMessage(message))
}
