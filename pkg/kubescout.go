package pkg

import (
	"fmt"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/diag"
	"github.com/reallyliri/kubescout/kubeclient"
	"github.com/reallyliri/kubescout/sink"
	"github.com/reallyliri/kubescout/store"
	log "github.com/sirupsen/logrus"
	"time"
)

// Scout the cluster for alerts. All parameters are optional, default values will be assigned, see CLI documentation.
func Scout(configuration *config.Config, alertSink sink.Sink) error {
	if alertSink == nil {
		outputMode := configuration.OutputMode
		switch outputMode {
		case "json":
			alertSink = &sink.JsonSink{}
		case "yaml":
			alertSink = &sink.YamlSink{}
		case "pretty":
			alertSink = &sink.PrettySink{}
		default:
			log.Errorf("output mode '%v' is not supported -- using pretty mode", outputMode)
			alertSink = &sink.PrettySink{}
		}
	}

	sto, err := store.LoadOrCreate(configuration)
	if err != nil {
		return err
	}

	client, err := kubeclient.CreateClient(configuration)

	if err != nil {
		return fmt.Errorf("failed to build kuberentes client: %v", err)
	}

	now := time.Now().UTC()
	err = diag.DiagnoseCluster(client, configuration, sto, now)

	if err != nil {
		return fmt.Errorf("failed to diagnose cluster: %v", err)
	}

	relevantMessages := sto.EntityAlerts()

	if len(relevantMessages) == 0 {
		return nil
	}

	alert := sink.Alert{
		ClusterName: configuration.ClusterName,
		Content:     relevantMessages,
	}
	err = alertSink.Report(alert)
	if err != nil {
		flushErr := sto.Flush(now)
		if flushErr != nil {
			log.Errorf("failed to flush to store: %v", flushErr)
		}
	}
	return err
}
