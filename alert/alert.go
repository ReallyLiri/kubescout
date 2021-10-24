package alert

import (
	"time"
)

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
