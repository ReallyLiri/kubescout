package config

import (
	"errors"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/fs"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	PodLogsTail                      int64
	EventsLimit                      int64
	KubeconfigFilePath               string
	TimeFormat                       string
	Locale                           *time.Location
	PodCreationGracePeriodSeconds    float64
	PodTerminationGracePeriodSeconds int64
	PodRestartGraceCount             int32
	NodeResourceUsageThreshold       float64
	ExcludeNamespaces                []string
	IncludeNamespaces                []string
	ClusterName                      string
	MessagesDeduplicationDuration    time.Duration
	StoreFilePath                    string
}

var Flags = []cli.Flag{
	&cli.BoolFlag{
		Name:     "verbose",
		Aliases:  []string{"vv"},
		Usage:    "Log verbose",
		Required: false,
		Value:    false,
	},
	&cli.Int64Flag{
		Name:     "logs-tail",
		Usage:    "Length of logs tail when reporting of a problematic pod's logs",
		Value:    250,
		Required: false,
	},
	&cli.Int64Flag{
		Name:     "events-limit",
		Usage:    "Limits of namespace events to fetch",
		Value:    150,
		Required: false,
	},
	&cli.StringFlag{
		Name:     "kubeconfig",
		Aliases:  []string{"c"},
		Usage:    "path to kubeconfig file, defaults to ~/.kube/config",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "time-format",
		Aliases:  []string{"f"},
		Value:    time.RFC822,
		Usage:    "format for printing timestamps",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "locale",
		Aliases:  []string{"l"},
		Value:    "UTC",
		Usage:    "localization to use when printing timestamps",
		Required: false,
	},
	&cli.Float64Flag{
		Name:     "pod-creation-grace-sec",
		Value:    30,
		Usage:    "grace time in seconds since pod creation",
		Required: false,
	},
	&cli.Int64Flag{
		Name:     "pod-termination-grace-sec",
		Value:    30,
		Usage:    "grace time in seconds since pod termination",
		Required: false,
	},
	&cli.IntFlag{
		Name:     "pod-restart-grace-count",
		Value:    3,
		Usage:    "grace time in seconds since pod termination",
		Required: false,
	},
	&cli.Float64Flag{
		Name:     "node-resource-usage-threshold",
		Value:    0.85,
		Usage:    "node resources usage threshold",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "exclude-ns",
		Aliases:  []string{"e"},
		Value:    "",
		Usage:    "namespaces to skip",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "include-ns",
		Aliases:  []string{"i"},
		Value:    "",
		Usage:    "namespaces to include (will skip any not listed if this option is used)",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "name",
		Aliases:  []string{"n"},
		Usage:    "name of the scouted cluster",
		Required: true,
	},
	&cli.IntFlag{
		Name:     "dedup-minutes",
		Aliases:  []string{"d"},
		Value:    60,
		Usage:    "number of minutes to silence duplicated or already observed alerts or 0 if this feature should not be applied",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "store-filepath",
		Aliases:  []string{"s"},
		Value:    "kube-scout.store.json",
		Usage:    "path to store file where duplicated message information will be persisted or empty string if this feature should not be applied",
		Required: false,
	},
}

func DefaultConfig() (*Config, error) {
	flagsSet, err := flagSet("default")
	if err != nil {
		return nil, err
	}
	config, err := ParseConfig(cli.NewContext(nil, flagsSet, nil))
	if err != nil {
		return nil, err
	}
	return config, nil
}

func flagSet(name string) (*flag.FlagSet, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, f := range Flags {
		if err := f.Apply(set); err != nil {
			return nil, fmt.Errorf("failed to build flag set due to flag %v: %v", f, err)
		}
	}
	return set, nil
}

func ParseConfig(c *cli.Context) (*Config, error) {
	config := &Config{
		PodLogsTail:                      c.Int64("logs-tail"),
		EventsLimit:                      c.Int64("events-limit"),
		KubeconfigFilePath:               c.String("kubeconfig"),
		TimeFormat:                       c.String("time-format"),
		PodCreationGracePeriodSeconds:    c.Float64("pod-creation-grace-sec"),
		PodTerminationGracePeriodSeconds: c.Int64("pod-termination-grace-sec"),
		PodRestartGraceCount:             int32(c.Int("pod-restart-grace-count")),
		NodeResourceUsageThreshold:       c.Float64("node-resource-usage-threshold"),
		ExcludeNamespaces:                splitListFlag(c.String("exclude-ns")),
		IncludeNamespaces:                splitListFlag(c.String("include-ns")),
		ClusterName:                      c.String("name"),
		MessagesDeduplicationDuration:    time.Minute * time.Duration(c.Int("dedup-minutes")),
		StoreFilePath:                    c.String("store-filepath"),
	}

	if config.KubeconfigFilePath == "" {
		homedirPath := homedir.HomeDir()
		if homedirPath == "" {
			return nil, fmt.Errorf("failed to determine homedir path")
		}
		config.KubeconfigFilePath = filepath.Join(homedirPath, ".kube/config")
	}
	if _, err := os.Stat(config.KubeconfigFilePath); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("kubeconfig does not exist at provided path '%v'", config.KubeconfigFilePath)
	}

	if config.StoreFilePath != "" {
		dirPath := filepath.Dir(config.StoreFilePath)
		err := validateDirectory(dirPath, true)
		if err != nil {
			return nil, fmt.Errorf("failed to create parent directories of store file at '%v': %v", dirPath, err)
		}
	}

	locationString := c.String("time-locale")
	location, err := time.LoadLocation(locationString)
	if err != nil {
		log.Printf("failed to parse locale '%v', using default - UTC", locationString)
		location = time.UTC
	}
	config.Locale = location

	log.SetFormatter(&log.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        config.TimeFormat,
		DisableLevelTruncation: true,
		PadLevelText:           true,
		QuoteEmptyFields:       true,
	})
	log.SetOutput(os.Stdout)
	if c.Bool("verbose") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	return config, nil
}
