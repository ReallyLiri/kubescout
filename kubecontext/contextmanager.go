package kubecontext

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"sort"
)

type ConfigContextManager interface {
	GetCurrentContext() string
	GetContextNames() ([]string, error)
	SetCurrentContext(name string) error
	TearDown() error
}

type configContextManager struct {
	filePath               string
	rootNode               *yaml.Node
	originalCurrentContext string
}

var _ ConfigContextManager = &configContextManager{}

func LoadKubeConfig(configFilePath string) (ConfigContextManager, error) {
	log.Infof("Using kubeconfig from '%v'", configFilePath)

	reader, err := os.OpenFile(configFilePath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("kubeconfig file not found at provided path '%v'", configFilePath)
		}
		return nil, fmt.Errorf("failed to open kubeconfig file at provided path '%v': %v", configFilePath, err)
	}

	defer func() {
		err := reader.Close()
		if err != nil {
			log.Errorf("failed to close kubeconfig file at '%v': %v", configFilePath, err)
		}
	}()

	var node yaml.Node
	if err = yaml.NewDecoder(reader).Decode(&node); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("yaml at %v is empty, please set up a valid kubeconfig", configFilePath)
		}
		return nil, fmt.Errorf("failed to decode '%v' as yaml: %v", configFilePath, err)
	}
	rootNode := node.Content[0]
	if rootNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("kubeconfig yaml root is %v and not map as expected", rootNode.Kind)
	}

	manager := &configContextManager{
		filePath: configFilePath,
		rootNode: rootNode,
	}
	manager.originalCurrentContext = manager.GetCurrentContext()

	return manager, err
}

func (manager *configContextManager) GetCurrentContext() string {
	node := manager.currentContextNode()
	if node == nil {
		return ""
	}
	return node.Value
}

func (manager *configContextManager) currentContextNode() *yaml.Node {
	node := valueOf(manager.rootNode, "current-context")
	if node == nil {
		return nil
	}
	return node
}

func (manager *configContextManager) createCurrentContextNode(name string) *yaml.Node {
	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "current-context",
		Tag:   "!!str",
	}
	valueNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: name,
		Tag:   "!!str",
	}
	manager.rootNode.Content = append(manager.rootNode.Content, keyNode, valueNode)
	return valueNode
}

func (manager *configContextManager) GetContextNames() (names []string, err error) {
	contexts := valueOf(manager.rootNode, "contexts")
	if contexts == nil || contexts.Kind != yaml.SequenceNode {
		return
	}

	for _, contextNode := range contexts.Content {
		name := valueOf(contextNode, "name")
		if name != nil {
			names = append(names, name.Value)
		}
	}

	sort.Strings(names)
	return
}

func (manager *configContextManager) SetCurrentContext(name string) error {
	log.Debugf("Setting current context to '%v'", name)
	node := manager.currentContextNode()
	if node == nil {
		node = manager.createCurrentContextNode(name)
	} else {
		node.Value = name
	}
	return manager.flush()
}

func (manager *configContextManager) TearDown() error {
	log.Debugf("Restoring current context to '%v'", manager.originalCurrentContext)
	return manager.SetCurrentContext(manager.originalCurrentContext)
}

func (manager *configContextManager) flush() error {
	writer, err := os.OpenFile(manager.filePath, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return fmt.Errorf("failed to open kubeconfig file at provided path for writing '%v': %v", manager.filePath, err)
	}

	defer func() {
		err := writer.Close()
		if err != nil {
			log.Errorf("failed to close kubeconfig file at '%v': %v", manager.filePath, err)
		}
	}()

	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(0)
	err = encoder.Encode(manager.rootNode)
	if err != nil {
		return fmt.Errorf("failed to encode kubeconfig yaml: %v", err)
	}

	return nil
}
