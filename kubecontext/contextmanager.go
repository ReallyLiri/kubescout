package kubecontext

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	originalCurrentContext string
	kubeconfig             *clientcmdapi.Config
}

var _ ConfigContextManager = &configContextManager{}

func LoadKubeConfig(configFilePath string) (ConfigContextManager, error) {
	log.Infof("Using kubeconfig from '%v'", configFilePath)

	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from '%v': %v", configFilePath, err)
	}

	manager := &configContextManager{
		filePath:               configFilePath,
		kubeconfig:             kubeconfig,
		originalCurrentContext: kubeconfig.CurrentContext,
	}
	manager.originalCurrentContext = manager.GetCurrentContext()

	return manager, err
}

func (manager *configContextManager) GetCurrentContext() string {
	return manager.kubeconfig.CurrentContext
}

func (manager *configContextManager) GetContextNames() (names []string, err error) {

	for name := range manager.kubeconfig.Contexts {
		if name != "" {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names, nil
}

func (manager *configContextManager) SetCurrentContext(name string) error {
	log.Debugf("Setting current context to '%v'", name)
	manager.kubeconfig.CurrentContext = name
	return manager.flush()
}

func (manager *configContextManager) TearDown() error {
	log.Debugf("Restoring current context to '%v'", manager.originalCurrentContext)
	return manager.SetCurrentContext(manager.originalCurrentContext)
}

func (manager *configContextManager) flush() error {
	err := clientcmd.WriteToFile(*manager.kubeconfig, manager.filePath)
	if err != nil {
		return fmt.Errorf("failed to write kubeconfig to '%v': %v", manager.filePath, err)
	}
	return nil
}
