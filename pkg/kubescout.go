package pkg

import (
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/internal/diag"
	"github.com/reallyliri/kubescout/internal/kubeclient"
	"github.com/reallyliri/kubescout/internal/kubeconfig"
	"github.com/reallyliri/kubescout/internal/store"
	"github.com/reallyliri/kubescout/sink"
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

	var contextNames []string
	var kconf kubeconfig.KubeConfig

	if cfg.RunningInCluster {
		contextNames = []string{"in-cluster"}
	} else {
		kconf, err = kubeconfig.LoadKubeconfig(cfg.KubeconfigFilePath)
		if err != nil {
			return err
		}

		contextNames, err = kubeconfig.ContextNames(
			kconf,
			cfg.ContextName,
			cfg.AllContexts,
			cfg.ExcludeContexts,
		)
		if err != nil {
			return err
		}
	}

	alerts := alert.NewAlerts()

	now := time.Now().UTC()

	var aggregatedErr error
	for i, contextName := range contextNames {
		if kconf != nil {
			kconf.CurrentContext = contextName
		}

		client, err := kubeclient.CreateClient(cfg, kconf)

		if err != nil {
			aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to build kuberentes client for %v: %v", contextName, err))
			continue
		}

		clusterStore := stor.GetClusterStore(contextName, now)

		log.Infof("Diagnosing cluster %v (%v/%v) ...", contextName, i+1, len(contextNames))

		err = diag.DiagnoseCluster(client, cfg, clusterStore, now)
		if err != nil {
			aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to diagnose cluster %v: %v", contextName, err))
			continue
		}

		clusterAlerts := clusterStore.Alerts
		sort.Sort(clusterAlerts)
		alerts.AddEntityAlerts(clusterAlerts)
	}

	if alerts.Empty() {
		return aggregatedErr
	}

	err = alertSink.Report(alerts)
	if err == nil {
		flushErr := stor.Flush(now)
		if flushErr != nil {
			aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to flush to store: %v", flushErr))
		}
	} else {
		aggregatedErr = multierr.Append(aggregatedErr, fmt.Errorf("failed to report alerts: %v", err))
	}

	return aggregatedErr
}
