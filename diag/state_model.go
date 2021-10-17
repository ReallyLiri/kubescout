package diag

import (
	"KubeScout/kubeclient"
	"fmt"
	"github.com/goombaio/orderedset"
	"strings"
)

type EntityState struct {
	FullName        string
	Kind            string
	messages        *orderedset.OrderedSet
	logsCollections map[string]string
	ActionTaken     bool
	client          kubeclient.KubernetesClient
}

func (state EntityState) Messages() []string {
	return toStrings(state.messages)
}

func (state EntityState) IsHealthy() bool {
	return state.messages.Empty()
}

func (state *EntityState) String() string {
	var builder strings.Builder
	if state.IsHealthy() {
		builder.WriteString(fmt.Sprintf("%v is healthy\n", state.FullName))
	} else {
		builder.WriteString(fmt.Sprintf("%v is un-healthy\n", state.FullName))
		for _, message := range state.Messages() {
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
	state.messages.Add(message)
}
