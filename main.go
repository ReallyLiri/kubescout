package main

import (
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"os"
)

const VERSION = "0.1.15"

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
			cfg, err := config.ParseConfig(ctx)
			if err != nil {
				return err
			}
			return pkg.Scout(cfg, nil)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("failed: %v", err)
		os.Exit(1)
	}
}
