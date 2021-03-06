package sink

import (
	"encoding/json"
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"go.uber.org/multierr"
	"gopkg.in/yaml.v3"
)

type Sink interface {
	Report(alerts *alert.Alerts) error
}

type JsonSink struct {
}

var _ Sink = &JsonSink{}

func (s JsonSink) Report(alerts *alert.Alerts) error {
	asJson, err := json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize alerts to json: %v", err)
	}
	fmt.Printf(string(asJson) + "\n")
	return nil
}

type YamlSink struct {
}

var _ Sink = &YamlSink{}

func (s YamlSink) Report(alerts *alert.Alerts) error {
	asYaml, err := yaml.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("failed to serialize alerts to yaml: %v", err)
	}
	fmt.Printf(string(asYaml) + "\n")
	return nil
}

type MultiSink struct {
	Sinks []Sink
}

var _ Sink = &MultiSink{}

func (s MultiSink) Report(alerts *alert.Alerts) (aggregatedErr error) {
	for _, sink := range s.Sinks {
		err := sink.Report(alerts)
		if err != nil {
			aggregatedErr = multierr.Append(aggregatedErr, err)
		}
	}
	return
}

type PrettySink struct {
}

var _ Sink = &PrettySink{}

func (s PrettySink) Report(alerts *alert.Alerts) error {
	fmt.Print(alerts.String())
	return nil
}

type DiscardSink struct {
}

var _ Sink = &DiscardSink{}

func (s DiscardSink) Report(_ *alert.Alerts) error {
	return nil
}
