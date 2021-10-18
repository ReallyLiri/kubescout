package kubeclient

import (
	"KubeScout/config"
	"bytes"
	"context"
	"fmt"
	"io"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"strings"
)

const PAGE_SIZE = 32

type KubernetesClient interface {
	GetNodes() ([]v1.Node, error)
	GetNamespaces() ([]v1.Namespace, error)
	GetPods(namespace string) ([]v1.Pod, error)
	GetReplicaSets(namespace string) ([]v12.ReplicaSet, error)
	GetPodLogs(namespace string, podName string, containerName string) (logs string, err error)
	GetEvents(namespace string) ([]v1.Event, error)
}

type remoteKubernetesClient struct {
	kubeClientSet *kubernetes.Clientset
	config        *config.Config
}

var _ KubernetesClient = &remoteKubernetesClient{}

func CreateClient(config *config.Config) (KubernetesClient, error) {

	kubeconfigFilePath := config.KubeconfigFilePath
	log.Printf("Building kuberenetes client from kubeconfig '%v'\n", kubeconfigFilePath)
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig from '%v': %v", kubeconfigFilePath, err)
	}

	clientSet, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	return &remoteKubernetesClient{
		kubeClientSet: clientSet,
		config:        config,
	}, nil
}

func (client *remoteKubernetesClient) GetNodes() ([]v1.Node, error) {

	var continueToken string
	var nodes = make([]v1.Node, 0)

	for {
		nodesData, err := client.kubeClientSet.CoreV1().Nodes().List(context.Background(), metaV1.ListOptions{
			Continue: continueToken,
			Limit:    PAGE_SIZE,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list nodes: %v", err)
		}
		nodes = append(nodes, nodesData.Items...)
		continueToken = nodesData.Continue
		if len(continueToken) == 0 {
			break
		}
	}

	return nodes, nil
}

func (client *remoteKubernetesClient) GetNamespaces() ([]v1.Namespace, error) {

	var continueToken string
	var namespaces = make([]v1.Namespace, 0)

	for {
		namespacesData, err := client.kubeClientSet.CoreV1().Namespaces().List(context.Background(), metaV1.ListOptions{
			Continue: continueToken,
			Limit:    PAGE_SIZE,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %v", err)
		}
		namespaces = append(namespaces, namespacesData.Items...)
		continueToken = namespacesData.Continue
		if len(continueToken) == 0 {
			break
		}
	}

	return namespaces, nil
}

func (client *remoteKubernetesClient) GetPods(namespace string) ([]v1.Pod, error) {

	var continueToken string
	var pods = make([]v1.Pod, 0)

	for {
		podsData, err := client.kubeClientSet.CoreV1().Pods(namespace).List(context.Background(), metaV1.ListOptions{
			Continue: continueToken,
			Limit:    PAGE_SIZE,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods in namespace '%v': %v", namespace, err)
		}
		pods = append(pods, podsData.Items...)
		continueToken = podsData.Continue
		if len(continueToken) == 0 {
			break
		}
	}

	return pods, nil
}

func (client *remoteKubernetesClient) GetReplicaSets(namespace string) ([]v12.ReplicaSet, error) {

	var continueToken string
	var replicaSets = make([]v12.ReplicaSet, 0)

	for {
		rsData, err := client.kubeClientSet.AppsV1().ReplicaSets(namespace).List(context.Background(), metaV1.ListOptions{
			Continue: continueToken,
			Limit:    PAGE_SIZE,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list replicaSets for namespace '%v': %v", namespace, err)
		}
		replicaSets = append(replicaSets, rsData.Items...)
		continueToken = rsData.Continue
		if len(continueToken) == 0 {
			break
		}
	}

	return replicaSets, nil
}

func (client *remoteKubernetesClient) GetPodLogs(namespace string, podName string, containerName string) (logs string, err error) {
	logsRequest := client.kubeClientSet.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		TailLines: &client.config.PodLogsTail,
		Container: containerName,
	})
	stream, err := logsRequest.Stream(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "waiting to start") {
			return "", nil
		}
		return "", fmt.Errorf("failed to stream logs of %v/%v/%v : %v", namespace, podName, containerName, err)
	}
	defer func() {
		err := stream.Close()
		if err != nil {
			log.Printf("failed to close stream for log request of %v/%v/%v: %v", namespace, podName, containerName, err)
		}
	}()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return "", fmt.Errorf("error in stream copy for %v/%v/%v : %v", namespace, podName, containerName, err)
	}
	return buf.String(), nil
}

func (client *remoteKubernetesClient) GetEvents(namespace string) ([]v1.Event, error) {
	eventsList, err := client.kubeClientSet.CoreV1().Events(namespace).List(context.Background(), metaV1.ListOptions{
		Limit: client.config.EventsLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events for %v: %v", namespace, err)
	}
	return eventsList.Items, nil
}
