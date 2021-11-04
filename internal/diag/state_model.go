package diag

import (
	"fmt"
	"github.com/goombaio/orderedmap"
	"github.com/reallyliri/kubescout/internal"
	"github.com/reallyliri/kubescout/internal/dedup"
	"github.com/reallyliri/kubescout/internal/store"
	"strings"
	"time"
)

type entityState struct {
	name             store.EntityName
	messages         []string
	node             string
	createdTimestamp time.Time
	logsCollections  map[string]string
}

type eventState struct {
	name          store.EntityName
	message       string
	lastTimestamp time.Time
}

func newState(entityName store.EntityName, createdTimestamp time.Time) *entityState {
	return &entityState{
		name:             entityName,
		messages:         []string{},
		createdTimestamp: createdTimestamp,
		logsCollections:  map[string]string{},
	}
}

func (state *entityState) isHealthy() bool {
	return len(state.messages) == 0
}

func (state *entityState) String() string {
	if state.isHealthy() {
		return fmt.Sprintf("%v is healthy\n", state.name.Name)
	}
	messages := append([]string{fmt.Sprintf("%v %v is un-healthy", state.name.Kind, state.name.Name)}, state.cleanMessages()...)
	return strings.Join(messages, "\n\t")
}

func (state *entityState) cleanMessages() []string {
	cleanMessages := orderedmap.NewOrderedMap()
	for _, message := range state.messages {
		a := dedup.CleanTemporal(message)
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
		return fmt.Sprintf("%v has a healthy event\n", state.name.Name)
	}
	return fmt.Sprintf("Event on %v %v: %v", state.name.Kind, state.name.Name, dedup.CleanTemporal(state.message))
}

func (state *eventState) cleanMessage() string {
	return dedup.CleanTemporal(state.message)
}
