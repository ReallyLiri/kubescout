package sink

import (
	"KubeScout/config"
	"cloud.google.com/go/kms/apiv1"
	"context"
	b64 "encoding/base64"
	"fmt"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"io/ioutil"
	"log"
	"net/http"
)

const encryptedReporterPasscodeBase64 = "CiQA0CSEiR5L/jIsWkX/jz8UTxOu3L1c8S4aN0H+euzZCXKhffISOgDkrAiL6dkunkKUBDI9ADQ4BcBxHm6KZvIg5rQlfPoopfi2BjDtJOipukk//GFWI6mFNRwAmmQ726Q="

func NewApiiroWebSink(config *config.Config) (Sink, error) {
	jsonLicense, err := readLicenseFile(config.ApiiroLicenseFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Apiiro license file from '%v': %v", config.ApiiroLicenseFilePath, err)
	}

	reporterPasscode, err := decryptPasscode(jsonLicense)
	if err != nil {
		return nil, err
	}

	return &webSink{
		postUrl:       "https://reporter.apiiro.com/api/messages/cluster",
		tlsSkipVerify: true,
		customizeRequest: func(request *http.Request) error {
			request.Header.Set("Authorization", string(reporterPasscode))
			return nil
		},
	}, nil
}

func readLicenseFile(licensePath string) ([]byte, error) {
	fileBytes, err := ioutil.ReadFile(licensePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read license file at '%v;: %v", licensePath, err)
	}

	bytesDecoded := make([]byte, len(fileBytes))
	_, err = b64.StdEncoding.Decode(bytesDecoded, fileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode licese file bytes at '%v': %v", licensePath, err)
	}

	return bytesDecoded, nil
}

func decryptPasscode(licenseJson []byte) ([]byte, error) {
	client, err := kms.NewKeyManagementClient(context.Background(), option.WithCredentialsJSON(licenseJson))
	if err != nil {
		return nil, fmt.Errorf("failed to create kms client: %v", err)
	}

	defer func() {
		err := client.Close()
		if err != nil {
			log.Printf("failed to close kms client: %v", err)
		}
	}()

	encryptedPasscode, err := b64.StdEncoding.DecodeString(encryptedReporterPasscodeBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 encrypted passcode: %v", err)
	}

	req := &kmspb.DecryptRequest{
		Name: fmt.Sprintf(
			"projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
			"apiiro",
			"global",
			"kube-scout",
			"kube-scout-key",
		),
		Ciphertext: encryptedPasscode,
	}

	response, err := client.Decrypt(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt passcode: %v", err)
	}

	return response.Plaintext, nil
}
