package sink

import (
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"strings"
)

type PrettySink struct {
}

var _ Sink = &PrettySink{}

func (s PrettySink) Report(alerts alert.Alerts) error {
	builder := strings.Builder{}
	for clusterName, clusterAlerts := range alerts.AlertsByClusterName {
		builder.WriteString(fmt.Sprintf("Found %v alerts for cluster %v:\n", len(clusterAlerts), clusterName))
		for _, entityAlert := range clusterAlerts {
			builder.WriteString(fmt.Sprintf("%v ", entityAlert.Kind))
			if entityAlert.Namespace != "" {
				builder.WriteString(fmt.Sprintf("%v/%v", entityAlert.Namespace, entityAlert.Name))
			} else {
				builder.WriteString(entityAlert.Name)
			}
			builder.WriteString("is un-healthy:\n")
			for _, message := range entityAlert.Messages {
				builder.WriteString(message)
				builder.WriteString("\n")
			}
			if len(entityAlert.Events) > 0 {
				for _, event := range entityAlert.Events {
					builder.WriteString(event)
					builder.WriteString("\n")
				}
			}
			if len(entityAlert.LogsByContainerName) > 0 {
				for containerName, logs := range entityAlert.LogsByContainerName {
					builder.WriteString(fmt.Sprintf("Logs of container %v:\n", containerName))
					builder.WriteString("--------\n")
					builder.WriteString(logs)
					builder.WriteString("\n--------\n")
				}
			}
			builder.WriteString("----------------\n")
		}
	}
	fmt.Print(builder.String())
	return nil
}
