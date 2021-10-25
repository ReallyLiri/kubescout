package diag

import (
	"fmt"
	"github.com/goombaio/orderedmap"
	"github.com/reallyliri/kubescout/internal"
	"strings"
	"time"
)

type entityState struct {
	name            string
	kind            string
	messages        []string
	logsCollections map[string]string
}

type eventState struct {
	involvedObject     string
	involvedObjectKind string
	message            string
	timestamp          time.Time
	namespace          string
}

func newState(name, kind string) *entityState {
	return &entityState{
		name:            name,
		kind:            kind,
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
	messages := append([]string{fmt.Sprintf("%v %v is un-healthy", state.kind, state.name)}, state.cleanMessages()...)
	return strings.Join(messages, "\n\t")
}

func (state *entityState) cleanMessages() []string {
	cleanMessages := orderedmap.NewOrderedMap()
	for _, message := range state.messages {
		a := cleanMessage(message)
		cleanMessages.Put(a, true)
	}
	return internal.CastToString(cleanMessages.Keys())
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
	state.messages = append(state.messages, message)
}

func (state *eventState) isHealthy() bool {
	return state.message == ""
}

func (state *eventState) String() string {
	if state.isHealthy() {
		return fmt.Sprintf("%v has a healthy event\n", state.involvedObject)
	}
	return fmt.Sprintf("Event on %v %v: %v", state.involvedObjectKind, state.involvedObject, cleanMessage(state.message))
}

func (state *eventState) cleanMessage() string {
	return cleanMessage(state.message)
}
