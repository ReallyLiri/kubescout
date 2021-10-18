package diag

import (
	"KubeScout/config"
	"KubeScout/kubeclient"
	"KubeScout/store"
	"fmt"
	"go.uber.org/multierr"
	"strings"
	"time"
)

const SEPARATOR = "------------------------------------------------"
const SUB_SEPERATOR = "------------------------"

type diagContext struct {
	config                *config.Config
	store                 *store.Store
	now                   time.Time
	includedNamespacesSet map[string]bool
	excludedNamespacesSet map[string]bool
}

func testContext() *diagContext {
	cfg, err := config.DefaultConfig()
	if err != nil {
		panic(err)
	}
	return &diagContext{config: cfg}
}

func (context *diagContext) handleState(state *entityState, printOnlyIfUnhealthy bool) (stored bool) {
	isHealthy := state.isHealthy()
	if !printOnlyIfUnhealthy || !isHealthy {
		fmt.Print(state)
	}
	if !isHealthy {
		builder := strings.Builder{}
		if state.kind != "Event" {
			builder.WriteString(fmt.Sprintf("%v %v is un-healthy", state.kind, state.fullName))
		}

		hashes := state.hashes.Values()
		var addedHashes []string
		for i, message := range state.messages {
			messageHash := hashes[i].(string)
			if context.store.ShouldAdd(messageHash, context.now) {
				addedHashes = append(addedHashes, messageHash)
				if builder.Len() > 0 {
					builder.WriteString("\n\t")
				}
				builder.WriteString(message)
			}
		}
		if len(addedHashes) == 0 {
			return false
		}
		if len(state.logsCollections) > 0 {
			builder.WriteString("\n")
			for container, logs := range state.logsCollections {
				builder.WriteString(fmt.Sprintf("logs of container %v:\n", container))
				builder.WriteString("<<<<<<<<<<\n")
				builder.WriteString(logs)
				builder.WriteString("\n>>>>>>>>>>")
			}
		}
		context.store.Add(builder.String(), addedHashes, context.now)
		return true
	}
	return false
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
	}

	cfg.Logger.Printf("Diagnosing cluster %v ...\n%v\n", cfg.ClusterName, SEPARATOR)

	nodes, err := client.GetNodes()
	if err != nil {
		aggregatedError = multierr.Append(aggregatedError, err)
	} else {
		cfg.Logger.Printf("Discovered %v nodes\n%v\n", len(nodes), SEPARATOR)
		for _, node := range nodes {
			nodeState, err := ctx.nodeState(&node, now, false)
			if err != nil {
				aggregatedError = multierr.Append(aggregatedError, err)
			} else {
				ctx.handleState(nodeState, false)
			}
		}
	}

	namespaces, err := client.GetNamespaces()
	if err != nil {
		aggregatedError = multierr.Append(aggregatedError, err)
		return
	}

	cfg.Logger.Printf("Discovered %v namespaces\n%v\n", len(namespaces), SEPARATOR)
	for _, namespace := range namespaces {
		namespaceName := namespace.Name
		if !ctx.isNamespaceRelevant(namespaceName) {
			continue
		}

		eventsByEntityName := make(map[string][]*entityState)

		events, err := client.GetEvents(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			cfg.Logger.Printf("Discovered %v events in namespace %v\n%v\n", len(events), namespaceName, SUB_SEPERATOR)
			for _, event := range events {
				eventState, err := ctx.eventState(&event, now)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					if event.InvolvedObject.Kind == "Node" {
						ctx.handleState(eventState, true)
					} else if !eventState.isHealthy() {
						eventsOfEntity, exists := eventsByEntityName[event.InvolvedObject.Name]
						if !exists {
							eventsOfEntity = []*entityState{}
						}
						eventsByEntityName[event.InvolvedObject.Name] = append(eventsOfEntity, eventState)
					}
				}
			}
		}

		pods, err := client.GetPods(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			cfg.Logger.Printf("Discovered %v pods in namespace %v\n%v\n", len(pods), namespaceName, SUB_SEPERATOR)
			for _, pod := range pods {
				podState, err := ctx.podState(&pod, now, client)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					stored := ctx.handleState(podState, false)
					if eventStates, found := eventsByEntityName[pod.Name]; found {
						if stored {
							for _, eventState := range eventStates {
								ctx.handleState(eventState, true)
							}
						}
						delete(eventsByEntityName, pod.Name)
					}
				}
			}
		}

		replicaSets, err := client.GetReplicaSets(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			cfg.Logger.Printf("Discovered %v replica sets in namespace %v\n%v\n", len(replicaSets), namespaceName, SUB_SEPERATOR)
			for _, replicaSet := range replicaSets {
				replicaSetState, err := ctx.replicaSetState(&replicaSet, now)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					stored := ctx.handleState(replicaSetState, true)
					if eventStates, found := eventsByEntityName[replicaSet.Name]; found {
						if stored {
							for _, eventState := range eventStates {
								ctx.handleState(eventState, true)
							}
						}
						delete(eventsByEntityName, replicaSet.Name)
					}
				}
			}
		}

		for _, eventStates := range eventsByEntityName {
			for _, entityState := range eventStates {
				ctx.handleState(entityState, true)
			}
		}
	}

	err = store.Flush()
	if err != nil {
		aggregatedError = multierr.Append(aggregatedError, err)
	}

	return
}
