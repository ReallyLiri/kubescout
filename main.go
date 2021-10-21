package main

import (
	"fmt"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/diag"
	"github.com/reallyliri/kubescout/kubeclient"
	"github.com/reallyliri/kubescout/sink"
	"github.com/reallyliri/kubescout/store"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"os"
	"time"
)

const VERSION = "0.1.1"

func main() {

	cli.AppHelpTemplate =
		`NAME:
   {{.Name}} - {{.Version}} - {{.Usage}}

USAGE:
   {{.Name}}{{range .Flags}}{{if and (not (eq .Name "help")) (not (eq .Name "version")) }} {{if .Required}}--{{.Name}} value{{end}}{{end}}{{end}} [optional flags]

OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}
`
	app := &cli.App{
		Name:    "kubescout",
		Usage:   "Scout for alarming issues in your Kubernetes cluster",
		Flags:   config.Flags,
		Version: VERSION,
		Action: func(ctx *cli.Context) error {
			configuration, err := config.ParseConfig(ctx)
			if err != nil {
				return err
			}
			return Scout(configuration, nil)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("failed: %v", err)
		os.Exit(1)
	}
}

// Scout the cluster for alerts. Args values should match CLI documentation.
func ScoutWithArgs(args []string, alertSink sink.Sink) error {
	configuration, err := config.FromArgs(args)
	if err != nil {
		return err
	}
	return Scout(configuration, alertSink)
}

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

	err = diag.DiagnoseCluster(client, configuration, sto, time.Now().UTC())

	if err != nil {
		return fmt.Errorf("failed to diagnose cluster: %v", err)
	}

	relevantMessages := sto.RelevantMessages()

	if len(relevantMessages) == 0 {
		return nil
	}

	alert := sink.Alert{
		ClusterName: configuration.ClusterName,
		Content:     relevantMessages,
	}
	return alertSink.Report(alert)
}
