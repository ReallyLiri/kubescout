package sink

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"strings"
)

type Alert struct {
	ClusterName string   `json:"cluster_name"`
	Content     []string `json:"content"`
}

type Sink interface {
	Report(message Alert) error
}

type JsonSink struct {
}

var _ Sink = &JsonSink{}

func (s JsonSink) Report(message Alert) error {
	asJson, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize message to json: %v", err)
	}
	fmt.Printf(string(asJson) + "\n")
	return nil
}

type YamlSink struct {
}

var _ Sink = &YamlSink{}

func (s YamlSink) Report(message Alert) error {
	asYaml, err := yaml.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize message to yaml: %v", err)
	}
	fmt.Printf(string(asYaml) + "\n")
	return nil
}

type PrettySink struct {
}

var _ Sink = &PrettySink{}

func (s PrettySink) Report(message Alert) error {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Found %v alerts for cluster %v:\n", len(message.Content), message.ClusterName))
	builder.WriteString(strings.Join(message.Content, "\n----------------\n"))
	builder.WriteString("\n")
	fmt.Print(builder.String())
	return nil
}
