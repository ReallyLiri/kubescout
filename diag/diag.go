package diag

import (
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/kubeclient"
	"github.com/reallyliri/kubescout/store"
	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"time"
)

type diagContext struct {
	config                *config.Config
	store                 *store.Store
	now                   time.Time
	includedNamespacesSet map[string]bool
	excludedNamespacesSet map[string]bool
	client                kubeclient.KubernetesClient
}

var diagKinds = map[string]bool{
	"Pod":        true,
	"Node":       true,
	"ReplicaSet": true,
}

func testContext() *diagContext {
	return testContextWithClient(nil)
}

func testContextWithClient(client kubeclient.KubernetesClient) *diagContext {
	cfg, err := config.DefaultConfig()
	if err != nil {
		panic(err)
	}
	log.SetLevel(log.DebugLevel)
	return &diagContext{config: cfg, client: client}
}

func (context *diagContext) handleEntityState(state *entityState, namespace string, events []*eventState) (stored bool) {
	isHealthy := state.isHealthy()
	if state.kind == "Node" && len(events) > 0 {
		isHealthy = false
	}
	if isHealthy {
		log.Tracef(state.String())
		return false
	}

	log.Infof(state.String())
	entityAlert := &alert.EntityAlert{
		ClusterName:         context.config.ClusterName,
		Namespace:           namespace,
		Name:                state.name,
		Kind:                state.kind,
		Messages:            []string{},
		Events:              []string{},
		LogsByContainerName: map[string]string{},
		Timestamp:           context.now,
	}

	hashes := state.hashes.Values()
	addedHashes := make(map[string]bool)
	for i, message := range state.messages {
		messageHash := hashes[i].(string)
		if !addedHashes[messageHash] && context.store.ShouldAdd(messageHash, context.now) {
			addedHashes[messageHash] = true
			entityAlert.Messages = append(entityAlert.Messages, message)
		}
	}
	for _, event := range events {
		if !addedHashes[event.hash] && context.store.ShouldAdd(event.hash, context.now) {
			addedHashes[event.hash] = true
			entityAlert.Events = append(entityAlert.Events, event.message)
		}
	}
	if len(addedHashes) == 0 {
		return false
	}
	entityAlert.LogsByContainerName = state.logsCollections

	context.store.Add(entityAlert, keys(addedHashes), context.now)
	return true
}

func (context *diagContext) handleStandaloneEvent(state *eventState) (stored bool) {
	isHealthy := state.isHealthy()
	if isHealthy {
		log.Tracef(state.String())
		return false
	}

	log.Infof(state.String())
	entityAlert := &alert.EntityAlert{
		ClusterName:         context.config.ClusterName,
		Namespace:           state.namespace,
		Name:                state.involvedObject,
		Kind:                state.involvedObjectKind,
		Messages:            []string{},
		Events:              []string{},
		LogsByContainerName: map[string]string{},
		Timestamp:           context.now,
	}

	if !context.store.ShouldAdd(state.hash, context.now) {
		return false
	}
	entityAlert.Events = append(entityAlert.Events, state.message)

	context.store.Add(entityAlert, []string{state.hash}, context.now)
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

func DiagnoseCluster(client kubeclient.KubernetesClient, cfg *config.Config, store *store.Store, now time.Time) (aggregatedError error) {
	ctx := diagContext{
		config:                cfg,
		store:                 store,
		now:                   now,
		includedNamespacesSet: toBoolMap(cfg.IncludeNamespaces),
		excludedNamespacesSet: toBoolMap(cfg.ExcludeNamespaces),
		client:                client,
	}

	log.Infof("Diagnosing cluster %v ...", cfg.ClusterName)

	eventsByEntityName := make(map[string][]*eventState)

	namespaces, err := client.GetNamespaces()
	if err != nil {
		aggregatedError = multierr.Append(aggregatedError, err)
		return
	}

	log.Infof("Discovered %v namespaces", len(namespaces))
	for _, namespace := range namespaces {
		namespaceName := namespace.Name
		if !ctx.isNamespaceRelevant(namespaceName) {
			continue
		}

		events, err := client.GetEvents(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Debugf("Discovered %v events in namespace %v", len(events), namespaceName)
			for _, event := range events {
				evState, err := ctx.eventState(&event, now)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else if !evState.isHealthy() {
					eventsOfEntity, exists := eventsByEntityName[event.InvolvedObject.Name]
					if !exists {
						eventsOfEntity = []*eventState{}
					}
					eventsByEntityName[event.InvolvedObject.Name] = append(eventsOfEntity, evState)
				}
			}
		}

		pods, err := client.GetPods(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Debugf("Discovered %v pods in namespace %v", len(pods), namespaceName)
			for _, pod := range pods {
				podState, err := ctx.podState(&pod, now)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					ctx.handleEntityState(podState, namespaceName, eventsByEntityName[pod.Name])
					delete(eventsByEntityName, pod.Name)
				}
			}
		}

		replicaSets, err := client.GetReplicaSets(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Debugf("Discovered %v replica sets in namespace %v", len(replicaSets), namespaceName)
			for _, replicaSet := range replicaSets {
				replicaSetState, err := ctx.replicaSetState(&replicaSet, now)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					ctx.handleEntityState(replicaSetState, "", eventsByEntityName[replicaSet.Name])
					delete(eventsByEntityName, replicaSet.Name)
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
			nodeState, err := ctx.nodeState(&node, now, false)
			if err != nil {
				aggregatedError = multierr.Append(aggregatedError, err)
			} else {
				ctx.handleEntityState(nodeState, "", eventsByEntityName[node.Name])
				delete(eventsByEntityName, node.Name)
			}
		}
	}

	for _, eventStates := range eventsByEntityName {
		for _, evState := range eventStates {
			if !diagKinds[evState.involvedObjectKind] {
				ctx.handleStandaloneEvent(evState)
			}
		}
	}

	return
}
