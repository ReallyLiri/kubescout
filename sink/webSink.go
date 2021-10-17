package sink

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type customizeRequest func(request *http.Request) error

type webSink struct {
	postUrl          string
	customizeRequest customizeRequest
	tlsSkipVerify    bool
}

var _ Sink = &webSink{}

func CreateWebSink(postUrl string) (Sink, error) {
	return &webSink{
		postUrl: postUrl,
	}, nil
}

func (sink webSink) Report(message Alert) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to serialize message to json: %v", err)
	}

	request, err := http.NewRequest("POST", sink.postUrl, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to construct http request to %v: %v", sink.postUrl, err)
	}

	if sink.customizeRequest != nil {
		err = sink.customizeRequest(request)
		if err != nil {
			return err
		}
	}

	client := http.DefaultClient

	if sink.tlsSkipVerify {
		skipVerifyTransport := http.DefaultTransport.(*http.Transport).Clone()
		skipVerifyTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = skipVerifyTransport
	}

	log.Printf("posting message to %v ...", sink.postUrl)
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to post message to %v: %v", sink.postUrl, err)
	}
	if response.StatusCode >= 400 {
		var responseBody []byte
		_, err := response.Body.Read(responseBody)
		if err == nil {
			log.Printf("failed to read response body: %v", err)
		}
		return fmt.Errorf("request to post message failed with code %v: %v", response.StatusCode, string(responseBody))
	}
	return nil
}
