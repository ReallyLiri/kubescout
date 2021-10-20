package sink

import (
	"encoding/json"
	"fmt"
)

type Alert struct {
	ClusterName string   `json:"cluster_name"`
	Content     []string `json:"content"`
}

type Sink interface {
	Report(message Alert) error
}

type LogSink struct {
}

func (s LogSink) Report(message Alert) error {
	asJson, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize message to json: %v", err)
	}
	fmt.Printf(string(asJson) + "\n")
	return nil
}

var _ Sink = &LogSink{}
