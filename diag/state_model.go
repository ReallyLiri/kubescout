package diag

import (
	"KubeScout/kubeclient"
	"fmt"
	"strings"
)

type EntityState struct {
	FullName           string
	Kind               string
	normalizedMessages map[string]bool
	Messages           []string
	logsCollections    map[string]string
	ActionTaken        bool
	client             kubeclient.KubernetesClient
}

func (state EntityState) IsHealthy() bool {
	return len(state.Messages) == 0
}

func (state *EntityState) String() string {
	var builder strings.Builder
	if state.IsHealthy() {
		builder.WriteString(fmt.Sprintf("%v is healthy\n", state.FullName))
	} else {
		builder.WriteString(fmt.Sprintf("%v %v is un-healthy\n", state.Kind, state.FullName))
		for _, message := range state.Messages {
			builder.WriteString(fmt.Sprintf("\t%v\n", message))
		}

	}
	return builder.String()
}

func (state *EntityState) appendMessage(format string, a ...interface{}) {
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
	state.Messages = append(state.Messages, cleanMessage(message))
}
