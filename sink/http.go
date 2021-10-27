package sink

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type TransportGetter func() (http.RoundTripper, error)

type CustomizeRequest func(request *http.Request) error

type ResponseVerify func(response *http.Response, responseBody string) error

func postHttp(url string, body []byte, transportGetter TransportGetter, customizeRequest CustomizeRequest, responseVerify ResponseVerify) (string, error) {
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

	if transportGetter != nil {
		client.Transport, err = transportGetter()
		if err != nil {
			return "", fmt.Errorf("failed to get http client transport: %v", err)
		}
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

	if responseVerify != nil {
		err = responseVerify(response, responseBodyString)
		if err != nil {
			return "", fmt.Errorf("response verification failed: %v", err)
		}
	}

	return responseBodyString, nil
}
