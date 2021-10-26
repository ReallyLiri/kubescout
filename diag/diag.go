package diag

import (
	"github.com/drgrib/iter"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/internal"
	"github.com/reallyliri/kubescout/kubeclient"
	"github.com/reallyliri/kubescout/store"
	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"time"
)

const sleepBetweenIterations = time.Second * time.Duration(3)

type diagContext struct {
	config                *config.Config
	store                 *store.ClusterStore
	now                   time.Time
	includedNamespacesSet map[string]bool
	excludedNamespacesSet map[string]bool
	client                kubeclient.KubernetesClient
	statesByName          map[entityName]*entityState
	eventsByName          map[entityName][]*eventState
}

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
		statesByName: map[entityName]*entityState{},
		eventsByName: map[entityName][]*eventState{},
		now:          now,
	}
}

func unhealthyEvents(events []*eventState) (unhealthy []*eventState) {
	for _, state := range events {
		if !state.isHealthy() {
			unhealthy = append(unhealthy, state)
		}
	}
	return
}

func (context *diagContext) handleEntityState(state *entityState, events []*eventState) (stored bool) {
	isHealthy := state.isHealthy()
	events = unhealthyEvents(events)
	if state.name.kind == "Node" && len(events) > 0 {
		isHealthy = false
	}
	if isHealthy {
		log.Trace(state.String())
		return false
	}

	entityAlert := &alert.EntityAlert{
		ClusterName:         context.store.Cluster,
		Namespace:           state.name.namespace,
		Name:                state.name.name,
		Kind:                state.name.kind,
		Messages:            []string{},
		Events:              []string{},
		LogsByContainerName: map[string]string{},
		Timestamp:           context.now,
	}

	addedHashes := make(map[string]bool)
	for _, message := range state.messages {
		messageHash := hash(state.name, normalizeMessage(message))
		if !addedHashes[messageHash] && context.store.ShouldAdd(messageHash, context.now) {
			addedHashes[messageHash] = true
			entityAlert.Messages = append(entityAlert.Messages, cleanMessage(message))
		}
	}
	for _, event := range events {
		messageHash := hash(event.name, normalizeMessage(event.message))
		if !addedHashes[messageHash] && context.store.ShouldAdd(messageHash, context.now) {
			addedHashes[messageHash] = true
			entityAlert.Events = append(entityAlert.Events, cleanMessage(event.message))
		}
	}

	deduped := len(addedHashes) == 0
	if !deduped {
		log.Infof("[DEDUPPED] %v", state)
	} else {
		log.Info(state.String())
		entityAlert.LogsByContainerName = state.logsCollections
		context.store.Add(entityAlert, internal.Keys(addedHashes), context.now)
	}

	return deduped
}

func (context *diagContext) handleStandaloneEvent(state *eventState) (stored bool) {
	isHealthy := state.isHealthy()
	if isHealthy {
		log.Tracef(state.String())
		return false
	}

	log.Infof(state.String())
	entityAlert := &alert.EntityAlert{
		ClusterName:         context.store.Cluster,
		Namespace:           state.name.namespace,
		Name:                state.name.name,
		Kind:                state.name.kind,
		Messages:            []string{},
		Events:              []string{},
		LogsByContainerName: map[string]string{},
		Timestamp:           context.now,
	}

	messageHash := hash(state.name, normalizeMessage(state.message))
	if !context.store.ShouldAdd(messageHash, context.now) {
		return false
	}
	entityAlert.Events = append(entityAlert.Events, cleanMessage(state.message))

	context.store.Add(entityAlert, []string{messageHash}, context.now)
	return true
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

func DiagnoseCluster(client kubeclient.KubernetesClient, cfg *config.Config, store *store.ClusterStore, now time.Time) (aggregatedError error) {
	context := diagContext{
		config:                cfg,
		store:                 store,
		now:                   now,
		includedNamespacesSet: internal.ToBoolMap(cfg.IncludeNamespaces),
		excludedNamespacesSet: internal.ToBoolMap(cfg.ExcludeNamespaces),
		client:                client,
		statesByName:          map[entityName]*entityState{},
		eventsByName:          map[entityName][]*eventState{},
	}

	log.Infof("Diagnosing cluster %v ...", store.Cluster)

	for i := range iter.N(cfg.Iterations) {
		if i > 0 {
			time.Sleep(sleepBetweenIterations)
		}
		err := context.clusterIteration(i)
		if err != nil {
			return err
		}
	}

	for name, state := range context.statesByName {
		context.handleEntityState(state, context.eventsByName[name])
		delete(context.eventsByName, name)
	}

	for _, states := range context.eventsByName {
		for _, state := range states {
			if !state.isHealthy() {
				context.handleStandaloneEvent(state)
			}
		}
	}

	return
}

func (context *diagContext) clusterIteration(i int) error {
	client := context.client
	namespaces, err := client.GetNamespaces()
	if err != nil {
		return err
	}

	var aggregatedError error

	log.Debugf("Discovered %v namespaces (iter=%v)", len(namespaces), i)
	for _, namespace := range namespaces {
		namespaceName := namespace.Name
		if !context.isNamespaceRelevant(namespaceName) {
			continue
		}

		events, err := client.GetEvents(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Debugf("Discovered %v events in namespace %v (iter=%v)", len(events), namespaceName, i)
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
			log.Debugf("Discovered %v pods in namespace %v (iter=%v)", len(pods), namespaceName, i)
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
			log.Debugf("Discovered %v replica sets in namespace %v (iter=%v)", len(replicaSets), namespaceName, i)
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
		log.Debugf("Discovered %v nodes (iter=%v)", len(nodes), i)
		for _, node := range nodes {
			_, err = context.nodeState(&node, false)
			if err != nil {
				aggregatedError = multierr.Append(aggregatedError, err)
			}
		}
	}

	return aggregatedError
}
