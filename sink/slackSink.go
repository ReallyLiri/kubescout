package sink

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

type slackSink struct {
	webhookUrl string
}

var _ Sink = &slackSink{}

func CreateSlackSink(webhookUrl string) (Sink, error) {
	return &slackSink{
		webhookUrl: webhookUrl,
	}, nil
}

func (sink slackSink) Report(alert Alert) error {
	builder := strings.Builder{}

	builder.WriteString(fmt.Sprintf("Alert for *%v*", alert.ClusterName))
	for _, line := range alert.Content {
		builder.WriteString(line)
	}

	buffer := bytes.NewBufferString(builder.String())
	var customizeRequest CustomizeRequest = func(request *http.Request) error {
		request.Header.Add("Content-Type", "application/json")
		return nil
	}

	responseBody, err := postHttp(sink.webhookUrl, buffer.Bytes(), customizeRequest, false)
	if err != nil {
		return err
	}
	if responseBody != "ok" {
		return fmt.Errorf("non-ok response from Slack: '%v'", responseBody)
	}

	return nil
}
