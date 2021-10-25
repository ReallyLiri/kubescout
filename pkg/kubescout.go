package pkg

import (
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/diag"
	"github.com/reallyliri/kubescout/kubeclient"
	"github.com/reallyliri/kubescout/kubecontext"
	"github.com/reallyliri/kubescout/sink"
	"github.com/reallyliri/kubescout/store"
	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"sort"
	"time"
)

// Scout the cluster for alerts. All parameters are optional, default values will be assigned, see CLI documentation.
func Scout(configuration *config.Config, alertSink sink.Sink) error {
	if alertSink == nil {
		alertSink = configuration.DefaultSink()
	}

	contextManager, err := kubecontext.LoadKubeConfig(configuration.KubeconfigFilePath)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	stor, err := store.LoadOrCreate(configuration)
	if err != nil {
		return fmt.Errorf("failed to load store: %v", err)
	}

	alerts := alert.NewAlerts()

	contextNames, err := configuration.ContextNames(contextManager)

	var aggregatedErr error
	for _, contextName := range contextNames {
		err = contextManager.SetCurrentContext(contextName)
		if err != nil {
			aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to set context to %v: %v", contextName, err))
		}

		client, err := kubeclient.CreateClient(configuration)

		if err != nil {
			aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to build kuberentes client for %v: %v", contextName, err))
		}

		clusterStore := stor.GetClusterStore(contextName, now)

		err = diag.DiagnoseCluster(client, configuration, clusterStore, now)
		if err != nil {
			aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to diagnose cluster %v: %v", contextName, err))
		}

		clusterAlerts := clusterStore.Alerts
		sort.Sort(clusterAlerts)
		alerts.AddEntityAlerts(clusterAlerts)
	}

	if aggregatedErr != nil {
		return aggregatedErr
	}

	if alerts.Empty() {
		return nil
	}

	err = alertSink.Report(alerts)
	if err == nil {
		flushErr := stor.Flush(now)
		if flushErr != nil {
			log.Errorf("failed to flush to store: %v", flushErr)
		}
	}
	return err
}
