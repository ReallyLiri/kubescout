package kubeclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"os"
)

type mockKubernetesClient struct {
	nodes       *v1.NodeList
	namespaces  *v1.NamespaceList
	pods        *v1.PodList
	replicaSets *v12.ReplicaSetList
	events      *v1.EventList
}

func (client *mockKubernetesClient) GetNodes() ([]v1.Node, error) {
	return client.nodes.Items, nil
}

func (client *mockKubernetesClient) GetNamespaces() ([]v1.Namespace, error) {
	return client.namespaces.Items, nil
}

func (client *mockKubernetesClient) GetPods(namespace string) ([]v1.Pod, error) {
	return client.pods.Items, nil
}

func (client *mockKubernetesClient) GetReplicaSets(namespace string) ([]v12.ReplicaSet, error) {
	return client.replicaSets.Items, nil
}

func (client *mockKubernetesClient) GetPodLogs(namespace string, podName string, containerName string) (string, error) {
	return fmt.Sprintf("%v/%v/%v/logs", namespace, podName, containerName), nil
}

func (client *mockKubernetesClient) GetEvents(namespace string) ([]v1.Event, error) {
	return client.events.Items, nil
}

var _ KubernetesClient = &mockKubernetesClient{}

func fromJson(filePath string, targetObject interface{}) error {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file at '%v': %v", filePath, err)
	}
	err = json.Unmarshal(bytes, targetObject)
	if err != nil {
		return fmt.Errorf("failed to deserialize json at '%v': %v", filePath, err)
	}
	return nil
}

func fileRelevant(filePath string) bool {
	if filePath == "" {
		return false
	}
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func CreateMockClient(
	nodesJsonFilePath string,
	namespacesJsonFilePath string,
	podsJsonFilePath string,
	rsJsonFilePath string,
	eventsFilePath string,
) (*mockKubernetesClient, error) {
	var err error
	client := &mockKubernetesClient{
		nodes:       &v1.NodeList{},
		namespaces:  &v1.NamespaceList{},
		pods:        &v1.PodList{},
		replicaSets: &v12.ReplicaSetList{},
		events:      &v1.EventList{},
	}
	if fileRelevant(nodesJsonFilePath) {
		err = fromJson(nodesJsonFilePath, &client.nodes)
		if err != nil {
			return nil, err
		}
	}
	if fileRelevant(namespacesJsonFilePath) {
		err = fromJson(namespacesJsonFilePath, &client.namespaces)
		if err != nil {
			return nil, err
		}
	}
	if fileRelevant(podsJsonFilePath) {
		err = fromJson(podsJsonFilePath, &client.pods)
		if err != nil {
			return nil, err
		}
	}
	if fileRelevant(rsJsonFilePath) {
		err = fromJson(rsJsonFilePath, &client.replicaSets)
		if err != nil {
			return nil, err
		}
	}
	if fileRelevant(eventsFilePath) {
		err = fromJson(eventsFilePath, &client.events)
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}
