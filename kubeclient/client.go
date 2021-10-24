package kubeclient

import (
	"bytes"
	"context"
	"fmt"
	"github.com/reallyliri/kubescout/config"
	log "github.com/sirupsen/logrus"
	"io"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

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
	log.Debugf("Building kubernetes client from kubeconfig '%v'\n", kubeconfigFilePath)
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
	var nodes []v1.Node
	err := pagedGet(
		nil,
		func(options metaV1.ListOptions) (runtime.Object, error) {
			newNodes, err := client.kubeClientSet.CoreV1().Nodes().List(context.Background(), options)
			if err != nil {
				return nil, fmt.Errorf("failed to list nodes: %v", err)
			}
			nodes = append(nodes, newNodes.Items...)
			return newNodes, nil
		},
	)
	return nodes, err
}

func (client *remoteKubernetesClient) GetNamespaces() ([]v1.Namespace, error) {
	var namespaces []v1.Namespace
	err := pagedGet(
		nil,
		func(options metaV1.ListOptions) (runtime.Object, error) {
			newNamespaces, err := client.kubeClientSet.CoreV1().Namespaces().List(context.Background(), options)
			if err != nil {
				return nil, fmt.Errorf("failed to list namespaces: %v", err)
			}
			namespaces = append(namespaces, newNamespaces.Items...)
			return newNamespaces, nil
		},
	)
	return namespaces, err
}

func (client *remoteKubernetesClient) GetPods(namespace string) ([]v1.Pod, error) {
	var pods []v1.Pod
	err := pagedGet(
		nil,
		func(options metaV1.ListOptions) (runtime.Object, error) {
			newPods, err := client.kubeClientSet.CoreV1().Pods(namespace).List(context.Background(), options)
			if err != nil {
				return nil, fmt.Errorf("failed to list pods in namespace '%v': %v", namespace, err)
			}
			pods = append(pods, newPods.Items...)
			return newPods, nil
		},
	)
	return pods, err
}

func (client *remoteKubernetesClient) GetReplicaSets(namespace string) ([]v12.ReplicaSet, error) {
	var replicaSets []v12.ReplicaSet
	err := pagedGet(
		nil,
		func(options metaV1.ListOptions) (runtime.Object, error) {
			newReplicaSets, err := client.kubeClientSet.AppsV1().ReplicaSets(namespace).List(context.Background(), options)
			if err != nil {
				return nil, fmt.Errorf("failed to list replicaSets for namespace '%v': %v", namespace, err)
			}
			replicaSets = append(replicaSets, newReplicaSets.Items...)
			return newReplicaSets, nil
		},
	)
	return replicaSets, err
}

func (client *remoteKubernetesClient) GetEvents(namespace string) ([]v1.Event, error) {
	listOptions := metaV1.ListOptions{
		Limit: client.config.EventsLimit,
	}
	var eventList []v1.Event
	err := pagedGet(
		&listOptions,
		func(options metaV1.ListOptions) (runtime.Object, error) {
			newEvents, err := client.kubeClientSet.CoreV1().Events(namespace).List(context.Background(), options)
			if err != nil {
				return nil, fmt.Errorf("failed to get events for %v: %v", namespace, err)
			}
			eventList = append(eventList, newEvents.Items...)
			return newEvents, nil
		},
	)
	return eventList, err
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
			log.Errorf("failed to close stream for log request of %v/%v/%v: %v", namespace, podName, containerName, err)
		}
	}()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return "", fmt.Errorf("error in stream copy for %v/%v/%v : %v", namespace, podName, containerName, err)
	}
	return buf.String(), nil
}
