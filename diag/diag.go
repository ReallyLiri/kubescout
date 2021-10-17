package diag

import (
	"KubeScout/config"
	"KubeScout/kubeclient"
	"KubeScout/store"
	"fmt"
	"github.com/urfave/cli/v2"
	"go.uber.org/multierr"
	"log"
	"strings"
	"time"
)

const SEPERATOR = "------------------------------------------------"
const SUB_SEPERATOR = "------------------------"

type diagContext struct {
	config                *config.Config
	store                 *store.Store
	now                   time.Time
	includedNamespacesSet map[string]bool
	excludedNamespacesSet map[string]bool
}

func testContext() *diagContext {
	flagsSet, err := config.FlagSet("test")
	if err != nil {
		panic(err)
	}
	cfg, err := config.ParseConfig(cli.NewContext(nil, flagsSet, nil))
	if err != nil {
		panic(err)
	}
	return &diagContext{config: cfg}
}

func (context *diagContext) handleState(state *EntityState, printOnlyIfUnhealthy bool) {
	isHealthy := state.IsHealthy()
	if !printOnlyIfUnhealthy || !isHealthy {
		fmt.Print(state)
	}
	if !isHealthy {
		message := state.String()
		messageHash := hash(message)
		if len(state.logsCollections) > 0 {
			builder := strings.Builder{}
			for container, logs := range state.logsCollections {
				builder.WriteString(fmt.Sprintf("logs of container %v:\n", container))
				builder.WriteString("<<<<<<<<<<\n")
				builder.WriteString(logs)
				builder.WriteString(">>>>>>>>>>")
			}
			message = message + builder.String()
		}
		context.store.TryAdd(messageHash, message, context.now)
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

func DiagnoseCluster(client kubeclient.KubernetesClient, cfg *config.Config, store *store.Store, now time.Time) (aggregatedError error) {
	ctx := diagContext{
		config:                cfg,
		store:                 store,
		now:                   now,
		includedNamespacesSet: toBoolMap(cfg.IncludeNamespaces),
		excludedNamespacesSet: toBoolMap(cfg.ExcludeNamespaces),
	}

	log.Printf("Diagnosing cluster %v ...\n%v\n", cfg.ClusterName, SEPERATOR)

	nodes, err := client.GetNodes()
	if err != nil {
		aggregatedError = multierr.Append(aggregatedError, err)
	} else {
		log.Printf("Disocvered %v nodes\n%v\n", len(nodes), SEPERATOR)
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

	log.Printf("Disocvered %v namespaces\n%v\n", len(namespaces), SEPERATOR)
	for _, namespace := range namespaces {
		namespaceName := namespace.Name
		if !ctx.isNamespaceRelevant(namespaceName) {
			continue
		}

		unhealthyEntities := map[string]bool{}

		pods, err := client.GetPods(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Printf("Disocvered %v pods in namespace %v\n%v\n", len(pods), namespaceName, SUB_SEPERATOR)
			for _, pod := range pods {
				podState, err := ctx.podState(&pod, now, client)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					ctx.handleState(podState, false)
					unhealthyEntities[pod.Name] = true
				}
			}
		}

		replicaSets, err := client.GetReplicaSets(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Printf("Disocvered %v replica sets in namespace %v\n%v\n", len(replicaSets), namespaceName, SUB_SEPERATOR)
			for _, replicaSet := range replicaSets {
				replicaSetState, err := ctx.replicaSetState(&replicaSet, now)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					ctx.handleState(replicaSetState, true)
					unhealthyEntities[replicaSet.Name] = true
				}
			}
		}

		events, err := client.GetEvents(namespaceName)
		if err != nil {
			aggregatedError = multierr.Append(aggregatedError, err)
		} else {
			log.Printf("Disocvered %v events in namespace %v\n%v\n", len(events), namespaceName, SUB_SEPERATOR)
			for _, event := range events {
				eventState, err := ctx.eventState(&event, now)
				if err != nil {
					aggregatedError = multierr.Append(aggregatedError, err)
				} else {
					if event.InvolvedObject.Kind == "Node" || unhealthyEntities[event.InvolvedObject.Name] {
						ctx.handleState(eventState, true)
					}
				}
			}
		}
	}

	err = store.Flush()
	if err != nil {
		aggregatedError = multierr.Append(aggregatedError, err)
	}

	return
}
