package sink

import (
	"encoding/json"
	"fmt"
	"github.com/reallyliri/kubescout/alert"
)

type webSink struct {
	postUrl          string
	customizeRequest CustomizeRequest
	responseVerify   ResponseVerify
	tlsSkipVerify    bool
}

var _ Sink = &webSink{}

func CreateWebSink(postUrl string, customizeRequest CustomizeRequest, responseVerify ResponseVerify, tlsSkipVerify bool) (Sink, error) {
	return &webSink{
		postUrl:          postUrl,
		customizeRequest: customizeRequest,
		responseVerify:   responseVerify,
		tlsSkipVerify:    tlsSkipVerify,
	}, nil
}

func (sink webSink) Report(alerts *alert.Alerts) error {
	body, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("failed to serialize alert to json: %v", err)
	}

	_, err = postHttp(sink.postUrl, body, sink.customizeRequest, sink.responseVerify, sink.tlsSkipVerify)

	return err
}
