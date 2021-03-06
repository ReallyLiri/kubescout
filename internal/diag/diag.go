package diag

import (
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/internal"
	"github.com/reallyliri/kubescout/internal/dedup"
	"github.com/reallyliri/kubescout/internal/kubeclient"
	"github.com/reallyliri/kubescout/internal/store"
	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"time"
)

type diagContext struct {
	config                *config.Config
	store                 *store.ClusterStore
	now                   time.Time
	includedNamespacesSet map[string]bool
	excludedNamespacesSet map[string]bool
	client                kubeclient.KubernetesClient
	statesByName          map[store.EntityName]*entityState
	eventsByName          map[store.EntityName][]*eventState
}

var excludeStandaloneEventsOnKinds = map[string]bool{
	"Pod":        true,
	"Node":       true,
	"ReplicaSet": true,
}

const graceTimeForEventSinceEntityCreation = time.Second * time.Duration(5)

func testContext(now time.Time) *diagContext {
	return testContextWithClient(now, nil)
}

func testContextWithClient(now time.Time, client kubeclient.KubernetesClient) *diagContext {
	cfg, err := config.DefaultConfig()
	if err != nil {
		panic(err)
	}
	log.SetLevel(log.DebugLevel)
	return &diagContext{
		config:       cfg,
		client:       client,
		statesByName: map[store.EntityName]*entityState{},
		eventsByName: map[store.EntityName][]*eventState{},
		now:          now,
	}
}

func unhealthyEvents(state *entityState, events []*eventState) (unhealthy []*eventState) {
	for _, evState := range events {
		if evState.isHealthy() {
			continue
		}
		if !evState.lastTimestamp.IsZero() && state != nil && !state.createdTimestamp.IsZero() {
			sinceCreation := evState.lastTimestamp.Sub(state.createdTimestamp)
			if sinceCreation < graceTimeForEventSinceEntityCreation {
				continue
			}
		}
		unhealthy = append(unhealthy, evState)
	}
	return
}

func (context *diagContext) handleEntityState(state *entityState, events []*eventState) {
	isHealthy := state.isHealthy()
	events = unhealthyEvents(state, events)
	if state.name.Kind == "Node" && len(events) > 0 {
		isHealthy = false
	}
	if isHealthy {
		log.Trace(state.String())
		return
	}

	entityAlert := &alert.EntityAlert{
		ClusterName:         context.store.Cluster,
		Namespace:           state.name.Namespace,
		Name:                state.name.Name,
		Kind:                state.name.Kind,
		Node:                state.node,
		Messages:            []string{},
		Events:              []string{},
		LogsByContainerName: map[string]string{},
		Timestamp:           context.now,
	}

	for _, message := range state.messages {
		stored := context.store.TryAdd(state.name, message, context.now)
		if stored {
			entityAlert.Messages = append(entityAlert.Messages, dedup.CleanTemporal(message))
		}
	}

	if len(entityAlert.Messages) == 0 {
		log.Infof("[DEDUPED] %v", state)
		return
	} else {
		setMinTimestamp(&entityAlert.Timestamp, state.problemTimestamp)
	}

	for _, event := range events {
		stored := context.store.TryAdd(state.name, event.message, context.now)
		if stored {
			entityAlert.Events = append(entityAlert.Events, dedup.CleanTemporal(event.message))
			setMinTimestamp(&entityAlert.Timestamp, event.firstTimestamp)
		}
	}

	log.Info(state.String())
	entityAlert.LogsByContainerName = state.logsCollections
	context.store.Alerts = append(context.store.Alerts, entityAlert)
}

func (context *diagContext) handleStandaloneEvents(name store.EntityName, events []*eventState) {

	events = unhealthyEvents(nil, events)

	if excludeStandaloneEventsOnKinds[name.Kind] {
		return
	}

	entityAlert := &alert.EntityAlert{
		ClusterName:         context.store.Cluster,
		Namespace:           name.Namespace,
		Name:                name.Name,
		Kind:                name.Kind,
		Messages:            []string{},
		Events:              []string{},
		LogsByContainerName: map[string]string{},
		Timestamp:           context.now,
	}

	for _, event := range events {
		stored := context.store.TryAdd(name, event.message, context.now)
		if stored {
			entityAlert.Events = append(entityAlert.Events, dedup.CleanTemporal(event.message))
			setMinTimestamp(&entityAlert.Timestamp, event.firstTimestamp)
		}
	}

	if len(entityAlert.Events) > 0 {
		context.store.Alerts = append(context.store.Alerts, entityAlert)
	}
}

func (context *diagContext) isNamespaceRelevant(namespaceName string) bool {
	if len(context.includedNamespacesSet) > 0 && !context.includedNamespacesSet[namespaceName] {
		return false
	}
	if len(context.excludedNamespacesSet) > 0 && context.excludedNamespacesSet[namespaceName] {
		return false
	}
	return true
}

func DiagnoseCluster(client kubeclient.KubernetesClient, cfg *config.Config, clusterStore *store.ClusterStore, now time.Time) (aggregatedError error) {
	context := diagContext{
		config:                cfg,
		store:                 clusterStore,
		now:                   now,
		includedNamespacesSet: internal.ToBoolMap(cfg.IncludeNamespaces),
		excludedNamespacesSet: internal.ToBoolMap(cfg.ExcludeNamespaces),
		client:                client,
		statesByName:          map[store.EntityName]*entityState{},
		eventsByName:          map[store.EntityName][]*eventState{},
	}

	err := context.collectStates()
	if err != nil {
		return err
	}

	for name, state := range context.statesByName {
		context.handleEntityState(state, context.eventsByName[name])
		delete(context.eventsByName, name)
	}

	for entityName, states := range context.eventsByName {
		context.handleStandaloneEvents(entityName, states)
	}

	return
}

func (context *diagContext) collectStates() error {
	client := context.client
	namespaces, err := client.GetNamespaces()
	if err != nil {
		return err
	}

	var aggregatedError error

	log.Debugf("Discovered %v namespaces", len(namespaces))
	for _, namespace := range namespaces {
		namespaceName := namespace.Name
		if !context.isNamespaceRelevant(namespaceName) {
			continue
		}

		events, err := client.GetEvents(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Debugf("Discovered %v events in namespace %v", len(events), namespaceName)
			for _, event := range events {
				_, err = context.eventState(&event)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				}
			}
		}

		pods, err := client.GetPods(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Debugf("Discovered %v pods in namespace %v", len(pods), namespaceName)
			for _, pod := range pods {
				_, err = context.podState(&pod)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				}
			}
		}

		replicaSets, err := client.GetReplicaSets(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Debugf("Discovered %v replica sets in namespace %v", len(replicaSets), namespaceName)
			for _, replicaSet := range replicaSets {
				_, err = context.replicaSetState(&replicaSet)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				}
			}
		}
	}

	nodes, err := client.GetNodes()
	if err != nil {
		aggregatedError = multierr.Append(aggregatedError, err)
	} else {
		log.Debugf("Discovered %v nodes", len(nodes))
		for _, node := range nodes {
			_, err = context.nodeState(&node, false)
			if err != nil {
				aggregatedError = multierr.Append(aggregatedError, err)
			}
		}
	}

	return aggregatedError
}
