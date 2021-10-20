package sink

import (
	"bytes"
	"crypto/tls"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type CustomizeRequest func(request *http.Request) error

func postHttp(url string, body []byte, customizeRequest CustomizeRequest, tlsSkipVerify bool) (string, error) {
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to construct http request to %v: %v", url, err)
	}

	if customizeRequest != nil {
		err = customizeRequest(request)
		if err != nil {
			return "", fmt.Errorf("failed to customize http request: %v", err)
		}
	}

	client := http.DefaultClient

	if tlsSkipVerify {
		skipVerifyTransport := http.DefaultTransport.(*http.Transport).Clone()
		skipVerifyTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		client.Transport = skipVerifyTransport
	}

	log.Debugf("posting to %v ...", url)
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to http post to %v: %v", url, err)
	}

	responseBody := new(bytes.Buffer)
	_, err = responseBody.ReadFrom(response.Body)
	if err != nil {
		log.Errorf("failed to read response body: %v", err)
	}

	responseBodyString := responseBody.String()
	if response.StatusCode >= 400 {
		return "", fmt.Errorf("request to post alert failed with code %v: %v", response.StatusCode, responseBodyString)
	}

	return responseBodyString, nil
}
