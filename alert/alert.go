package alert

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

var kindToOrder = map[string]int{
	"Node":       1,
	"Namespace":  2,
	"ReplicaSet": 3,
	"Pod":        4,
}

type EntityAlert struct {
	ClusterName         string            `json:"cluster_name"`
	Namespace           string            `json:"namespace"`
	Name                string            `json:"name"`
	Kind                string            `json:"kind"`
	Messages            []string          `json:"messages"`
	Events              []string          `json:"events"`
	LogsByContainerName map[string]string `json:"logs_by_container_name"`
	Timestamp           time.Time         `json:"timestamp"`
}

type Alerts struct {
	AlertsByClusterName map[string]EntityAlerts `json:"alerts_by_cluster_name"`
}

func NewAlerts() *Alerts {
	return &Alerts{AlertsByClusterName: map[string]EntityAlerts{}}
}

func (alerts *Alerts) AddEntityAlerts(entityAlerts EntityAlerts) {
	alertsMap := alerts.AlertsByClusterName
	for _, alert := range entityAlerts {
		if _, found := alertsMap[alert.ClusterName]; !found {
			alertsMap[alert.ClusterName] = EntityAlerts{}
		}
		alertsMap[alert.ClusterName] = append(alertsMap[alert.ClusterName], alert)
	}
}

func (alerts *Alerts) Empty() bool {
	for _, entityAlerts := range alerts.AlertsByClusterName {
		if len(entityAlerts) > 0 {
			return false
		}
	}
	return true
}

type EntityAlerts []*EntityAlert

var _ sort.Interface = &EntityAlerts{}

func (alerts EntityAlerts) Len() int {
	return len(alerts)
}

func (alerts EntityAlerts) Less(i, j int) bool {
	kind1, found1 := kindToOrder[alerts[i].Kind]
	kind2, found2 := kindToOrder[alerts[j].Kind]
	if found1 == found2 {
		if kind1 == kind2 {
			return alerts[i].Name < alerts[j].Name
		}
		return kind1 < kind2
	}
	return found1
}

func (alerts EntityAlerts) Swap(i, j int) {
	tmp := alerts[i]
	alerts[i] = alerts[j]
	alerts[j] = tmp
}

func (alerts *Alerts) String() string {
	builder := strings.Builder{}
	for clusterName, clusterAlerts := range alerts.AlertsByClusterName {
		builder.WriteString(fmt.Sprintf("Found %v alerts for cluster %v:\n", len(clusterAlerts), clusterName))
		for _, entityAlert := range clusterAlerts {
			builder.WriteString(entityAlert.String())
			builder.WriteString("\n----------------\n")
		}
	}
	return builder.String()
}

func (entityAlert *EntityAlert) String() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%v ", entityAlert.Kind))
	if entityAlert.Namespace != "" {
		builder.WriteString(fmt.Sprintf("%v/%v", entityAlert.Namespace, entityAlert.Name))
	} else {
		builder.WriteString(entityAlert.Name)
	}
	builder.WriteString(" is un-healthy:")
	for _, message := range entityAlert.Messages {
		builder.WriteString("\n")
		builder.WriteString(message)
	}
	if len(entityAlert.Events) > 0 {
		for _, event := range entityAlert.Events {
			builder.WriteString("\n")
			builder.WriteString(event)
		}
	}
	if len(entityAlert.LogsByContainerName) > 0 {
		for containerName, logs := range entityAlert.LogsByContainerName {
			builder.WriteString("\n")
			builder.WriteString(fmt.Sprintf("Logs of container %v:\n", containerName))
			builder.WriteString("--------\n")
			builder.WriteString(logs)
			builder.WriteString("\n--------")
		}
	}
	return builder.String()
}
