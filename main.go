package main

import (
	"KubeScout/config"
	"KubeScout/diag"
	"KubeScout/kubeclient"
	"KubeScout/sink"
	"KubeScout/store"
	"fmt"
	"github.com/urfave/cli/v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"log"
	"os"
	"time"
)

const VERSION = "0.1.0"

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

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)
	app := &cli.App{
		Name:    "kubescout",
		Usage:   "Scout for alarming issues in your Kubernetes cluster",
		Flags:   config.Flags,
		Version: VERSION,
		Action: func(ctx *cli.Context) error {
			cfg, err := config.ParseConfig(ctx)
			if err != nil {
				return err
			}

			sto, err := store.LoadOrCreate(cfg)
			if err != nil {
				return err
			}

			var outputSink sink.Sink = sink.LogSink{}
			if cfg.ApiiroLicenseFilePath != "" {
				apiiroSink, err := sink.NewApiiroWebSink(cfg)
				if err != nil {
					log.Printf("failed to initialize Apiiro web sink: %v", err)
				} else {
					outputSink = apiiroSink
				}
			}

			client, err := kubeclient.CreateClient(cfg)

			if err != nil {
				return fmt.Errorf("failed to build kuberentes client: %v", err)
			}

			err = diag.DiagnoseCluster(client, cfg, sto, time.Now().UTC())

			if err != nil {
				return fmt.Errorf("failed to diagnose cluster: %v", err)
			}

			relevantMessages := sto.RelevantMessages()

			if len(relevantMessages) == 0 {
				return nil
			}

			alert := sink.Alert{
				CustomerName: cfg.ClusterName,
				Content:      relevantMessages,
			}
			return outputSink.Report(alert)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Printf("failed: %v", err)
		os.Exit(1)
	}
}
