package sink

import (
	"encoding/json"
	"fmt"
)

type webSink struct {
	postUrl          string
	customizeRequest CustomizeRequest
	tlsSkipVerify    bool
}

var _ Sink = &webSink{}

func CreateWebSink(postUrl string, customizeRequest CustomizeRequest, tlsSkipVerify bool) (Sink, error) {
	return &webSink{
		postUrl:          postUrl,
		customizeRequest: customizeRequest,
		tlsSkipVerify:    tlsSkipVerify,
	}, nil
}

func (sink webSink) Report(alert Alert) error {
	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to serialize alert to json: %v", err)
	}

	_, err = postHttp(sink.postUrl, body, sink.customizeRequest, sink.tlsSkipVerify)

	return err
}
