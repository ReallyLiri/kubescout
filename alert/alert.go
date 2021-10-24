package alert

import (
	"sort"
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
	AlertsByClusterName map[string][]EntityAlert `json:"alerts_by_cluster_name"`
}

func NewAlert() *Alerts {
	return &Alerts{AlertsByClusterName: map[string][]EntityAlert{}}
}

func (alerts *Alerts) AddEntityAlerts(entityAlerts ...EntityAlert) {
	alertsMap := alerts.AlertsByClusterName
	for _, alert := range entityAlerts {
		if _, found := alertsMap[alert.ClusterName]; !found {
			alertsMap[alert.ClusterName] = []EntityAlert{}
		}
		alertsMap[alert.ClusterName] = append(alertsMap[alert.ClusterName], alert)
	}
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
