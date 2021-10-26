package pkg

import (
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/diag"
	"github.com/reallyliri/kubescout/kubeclient"
	"github.com/reallyliri/kubescout/kubeconfig"
	"github.com/reallyliri/kubescout/sink"
	"github.com/reallyliri/kubescout/store"
	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"sort"
	"time"
)

// Scout the cluster for alerts. All parameters are optional, default values will be assigned, see CLI documentation.
func Scout(cfg *config.Config, alertSink sink.Sink) error {
	if alertSink == nil {
		alertSink = cfg.DefaultSink()
	}

	stor, err := store.LoadOrCreate(cfg)
	if err != nil {
		return err
	}

	kconf, err := kubeconfig.LoadKubeconfig(cfg.KubeconfigFilePath)
	if err != nil {
		return err
	}

	contextNames, err := kubeconfig.ContextNames(
		kconf,
		cfg.ContextName,
		cfg.AllContexts,
		cfg.ExcludeContexts,
	)
	if err != nil {
		return err
	}

	alerts := alert.NewAlerts()

	now := time.Now().UTC()

	var aggregatedErr error
	for _, contextName := range contextNames {
		kconf.CurrentContext = contextName

		client, err := kubeclient.CreateClient(cfg, kconf)

		if err != nil {
			aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to build kuberentes client for %v: %v", contextName, err))
		}

		clusterStore := stor.GetClusterStore(contextName, now)

		err = diag.DiagnoseCluster(client, cfg, clusterStore, now)
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
