package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/reallyliri/kubescout/internal/kubeconfig"
	"github.com/reallyliri/kubescout/sink"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	PodLogsTail                      int64
	EventsLimit                      int64
	KubeconfigFilePath               string
	RunningInCluster                 bool
	TimeFormat                       string
	Locale                           *time.Location
	PodCreationGracePeriodSeconds    float64
	PodStartingGracePeriodSeconds    float64
	PodTerminationGracePeriodSeconds int64
	PodRestartGraceCount             int32
	NodeResourceUsageThreshold       float64
	ExcludeNamespaces                []string
	IncludeNamespaces                []string
	MessagesDeduplicationDuration    time.Duration
	StoreFilePath                    string
	OutputMode                       string
	ContextName                      string
	AllContexts                      bool
	ExcludeContexts                  []string
	NotInCluster                     bool
}

var Flags = []cli.Flag{
	&cli.BoolFlag{
		Name:     "verbose",
		Aliases:  []string{"vv"},
		Usage:    "Verbose logging",
		Required: false,
		Value:    false,
		EnvVars:  []string{"VERBOSE"},
	},
	&cli.Int64Flag{
		Name:     "logs-tail",
		Usage:    "Specifies the logs tail length when reporting logs from a problematic pod, use 0 to disable log extraction",
		Value:    250,
		Required: false,
		EnvVars:  []string{"LOGS_TAIL"},
	},
	&cli.Int64Flag{
		Name:     "events-limit",
		Usage:    "Maximum number of namespace events to fetch",
		Value:    150,
		Required: false,
		EnvVars:  []string{"EVENTS_LIMIT"},
	},
	&cli.StringFlag{
		Name:     "kubeconfig",
		Aliases:  []string{"k"},
		Usage:    "kubeconfig file path, defaults to env var KUBECONFIG or ~/.kube/config, can be omitted when running in cluster",
		Required: false,
		EnvVars:  []string{"KUBECONFIG"},
	},
	&cli.StringFlag{
		Name:     "time-format",
		Aliases:  []string{"f"},
		Value:    time.RFC822,
		Usage:    "timestamp print format",
		Required: false,
		EnvVars:  []string{"TIME_FORMAT"},
	},
	&cli.StringFlag{
		Name:     "locale",
		Aliases:  []string{"l"},
		Value:    "UTC",
		Usage:    "timestamp print localization",
		Required: false,
		EnvVars:  []string{"LOCALE"},
	},
	&cli.Float64Flag{
		Name:     "pod-creation-grace-sec",
		Value:    5,
		Usage:    "grace period in seconds since pod creation before checking its statuses",
		Required: false,
		EnvVars:  []string{"POD_CREATION_GRACE_SEC"},
	},
	&cli.Int64Flag{
		Name:     "pod-starting-grace-sec",
		Value:    600,
		Usage:    "grace period in seconds since pod creation before alarming on non running states",
		Required: false,
		EnvVars:  []string{"POD_STARTING_GRACE_SEC"},
	},
	&cli.Int64Flag{
		Name:     "pod-termination-grace-sec",
		Value:    60,
		Usage:    "grace period in seconds since pod termination",
		Required: false,
		EnvVars:  []string{"POD_TERMINATION_GRACE_SEC"},
	},
	&cli.IntFlag{
		Name:     "pod-restart-grace-count",
		Value:    3,
		Usage:    "grace count for pod restarts",
		Required: false,
		EnvVars:  []string{"POD_RESTART_GRACE_COUNT"},
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
		EnvVars:  []string{"EXCLUDE_NS"},
	},
	&cli.StringFlag{
		Name:     "include-ns",
		Aliases:  []string{"n"},
		Value:    "",
		Usage:    "namespaces to include (will skip any not listed if this option is used)",
		Required: false,
		EnvVars:  []string{"INCLUDE_NS"},
	},
	&cli.IntFlag{
		Name:     "dedup-minutes",
		Aliases:  []string{"d"},
		Value:    60,
		Usage:    "time in minutes to silence duplicate or already observed alerts, or 0 to disable deduplication",
		Required: false,
		EnvVars:  []string{"DEDUP_MINUTES"},
	},
	&cli.StringFlag{
		Name:     "store-filepath",
		Aliases:  []string{"s"},
		Value:    "kube-scout.store.json",
		Usage:    "path to store file where state will be persisted or empty string to disable persistency",
		Required: false,
		EnvVars:  []string{"STORE_FILEPATH"},
	},
	&cli.StringFlag{
		Name:     "output",
		Aliases:  []string{"o"},
		Value:    "pretty",
		Usage:    "output mode, one of pretty/json/yaml/discard",
		Required: false,
		EnvVars:  []string{"OUTPUT_MODE"},
	},
	&cli.StringFlag{
		Name:     "context",
		Aliases:  []string{"c"},
		Value:    "",
		Usage:    "context name to use from kubeconfig, defaults to current context",
		Required: false,
	},
	&cli.BoolFlag{
		Name:     "not-in-cluster",
		Value:    false,
		Usage:    "hint to scan out of cluster even if technically kubescout is running in a pod",
		Required: false,
		EnvVars:  []string{"NOT_IN_CLUSTER"},
	},
	&cli.BoolFlag{
		Name:     "all-contexts",
		Aliases:  []string{"a"},
		Value:    false,
		Usage:    "iterate all kubeconfig contexts, 'context' flag will be ignored if this flag is set",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "exclude-contexts",
		Value:    "",
		Usage:    "a comma separated list of kubeconfig context names to skip, only relevant if 'all-contexts' flag is set",
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

func FromArgs(args []string) (config *Config, err error) {
	app := &cli.App{
		Flags: Flags,
		Action: func(ctx *cli.Context) error {
			config, err = ParseConfig(ctx)
			return nil
		},
	}
	runErr := app.Run(args)
	if err == nil {
		err = runErr
	}
	return
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
		PodStartingGracePeriodSeconds:    c.Float64("pod-starting-grace-sec"),
		PodTerminationGracePeriodSeconds: c.Int64("pod-termination-grace-sec"),
		PodRestartGraceCount:             int32(c.Int("pod-restart-grace-count")),
		NodeResourceUsageThreshold:       c.Float64("node-resource-usage-threshold"),
		ExcludeNamespaces:                splitListFlag(c.String("exclude-ns")),
		IncludeNamespaces:                splitListFlag(c.String("include-ns")),
		MessagesDeduplicationDuration:    time.Minute * time.Duration(c.Int("dedup-minutes")),
		StoreFilePath:                    c.String("store-filepath"),
		OutputMode:                       c.String("output"),
		ContextName:                      c.String("context"),
		AllContexts:                      c.Bool("all-contexts"),
		ExcludeContexts:                  splitListFlag(c.String("exclude-contexts")),
		NotInCluster:                     c.Bool("not-in-cluster"),
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
		log.SetLevel(log.TraceLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if config.KubeconfigFilePath == "" {
		config.KubeconfigFilePath, config.RunningInCluster, err = kubeconfig.DefaultKubeconfigPath(config.NotInCluster)
		if err != nil || (config.KubeconfigFilePath == "" && !config.RunningInCluster) {
			return nil, fmt.Errorf("failed to determine default kubeconfig file path: %v", err)
		}
	}

	if log.GetLevel() >= log.DebugLevel {
		configJson, err := json.MarshalIndent(config, "", " ")
		if err != nil {
			log.Errorf("failed to serialize config to json: %v", err)
		}
		log.Debugf("Loaded config:\n%v", string(configJson))
	}

	return config, nil
}

func (config *Config) DefaultSink() sink.Sink {
	switch config.OutputMode {
	case "json":
		return &sink.JsonSink{}
	case "yaml":
		return &sink.YamlSink{}
	case "pretty":
		return &sink.PrettySink{}
	case "discard":
		return &sink.DiscardSink{}
	default:
		log.Errorf("output mode '%v' is not supported -- using pretty mode", config.OutputMode)
		return &sink.PrettySink{}
	}
}
