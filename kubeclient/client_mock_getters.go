package kubeclient

import (
	"github.com/stretchr/testify/require"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"path"
	"runtime"
	"testing"
)

var apiResponsesDirectoryPath string

func init() {
	_, filePath, _, _ := runtime.Caller(0)
	apiResponsesDirectoryPath = path.Join(filePath, "../../test-resources/api-responses")
}

func GetEvents(t *testing.T, fileName string) ([]v1.Event, error) {
	client, err := CreateMockClient(
		"",
		"",
		"",
		"",
		path.Join(apiResponsesDirectoryPath, "get-events", fileName),
	)
	require.Nil(t, err)
	require.NotNil(t, client)

	events, err := client.GetEvents("")
	return events, err
}

func GetNodes(t *testing.T, fileName string) ([]v1.Node, error) {
	client, err := CreateMockClient(
		path.Join(apiResponsesDirectoryPath, "get-nodes", fileName),
		"",
		"",
		"",
		"",
	)
	require.Nil(t, err)
	require.NotNil(t, client)

	nodes, err := client.GetNodes()
	return nodes, err
}

func GetNamespaces(t *testing.T, fileName string) ([]v1.Namespace, error) {
	client, err := CreateMockClient(
		"",
		path.Join(apiResponsesDirectoryPath, "get-ns", fileName),
		"",
		"",
		"",
	)
	require.Nil(t, err)
	require.NotNil(t, client)

	namespaces, err := client.GetNamespaces()
	return namespaces, err
}

func GetPods(t *testing.T, fileName string) ([]v1.Pod, error) {
	client, err := CreateMockClient(
		"",
		"",
		path.Join(apiResponsesDirectoryPath, "get-pods", fileName),
		"",
		"",
	)
	require.Nil(t, err)
	require.NotNil(t, client)

	pods, err := client.GetPods("")
	return pods, err
}

func GetReplicaSets(t *testing.T, fileName string) ([]v12.ReplicaSet, error) {
	client, err := CreateMockClient(
		"",
		"",
		"",
		path.Join(apiResponsesDirectoryPath, "get-rs", fileName),
		"",
	)
	require.Nil(t, err)
	require.NotNil(t, client)

	replicaSets, err := client.GetReplicaSets("")
	return replicaSets, err
}
