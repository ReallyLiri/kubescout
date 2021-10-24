package diag

import (
	"fmt"
	"github.com/goombaio/orderedset"
	"strings"
	"time"
)

type entityState struct {
	name            string
	kind            string
	hashes          *orderedset.OrderedSet
	messages        []string
	logsCollections map[string]string
}

type eventState struct {
	involvedObject     string
	involvedObjectKind string
	hash               string
	message            string
	timestamp          time.Time
	namespace          string
}

func newState(name, kind string) *entityState {
	return &entityState{
		name:            name,
		kind:            kind,
		hashes:          orderedset.NewOrderedSet(),
		messages:        []string{},
		logsCollections: map[string]string{},
	}
}

func (state *entityState) isHealthy() bool {
	return len(state.messages) == 0
}

func (state *entityState) String() string {
	if state.isHealthy() {
		return fmt.Sprintf("%v is healthy\n", state.name)
	}
	messages := append([]string{fmt.Sprintf("%v %v is un-healthy", state.kind, state.name)}, state.messages...)
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
	messageHash := hash(state.name, normalizeMessage(message))
	if state.hashes.Contains(messageHash) {
		return
	}
	state.hashes.Add(messageHash)
	state.messages = append(state.messages, cleanMessage(message))
}

func (state *eventState) isHealthy() bool {
	return state.message == ""
}

func (state *eventState) String() string {
	if state.isHealthy() {
		return fmt.Sprintf("%v has a healthy event\n", state.involvedObject)
	}
	return fmt.Sprintf("Event on %v %v: %v", state.involvedObjectKind, state.involvedObject, state.message)
}
