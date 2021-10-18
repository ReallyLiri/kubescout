package diag

import (
	"KubeScout/kubeclient"
	"fmt"
	"strings"
)

type entityState struct {
	fullName           string
	kind               string
	normalizedMessages map[string]bool
	messages           []string
	logsCollections    map[string]string
	client             kubeclient.KubernetesClient
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
		messages = append([]string{fmt.Sprintf("%v %v is un-healthy", state.kind, state.fullName)}, state.messages...)
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
	normalizedMessage := normalizeMessage(message)
	if _, found := state.normalizedMessages[normalizedMessage]; found {
		return
	}
	state.normalizedMessages[normalizedMessage] = true
	state.messages = append(state.messages, cleanMessage(message))
}
