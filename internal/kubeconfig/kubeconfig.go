package kubeconfig

import (
	"fmt"
	"github.com/reallyliri/kubescout/internal"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sort"
)

const serviceAccountTokenInClusterPath = "/var/run/secrets/kubernetes.io/serviceaccount"

type KubeConfig *clientcmdapi.Config

func DefaultKubeconfigPath(forceNotInCluster bool) (filePath string, runningInCluster bool, err error) {
	if !forceNotInCluster {
		if _, err = os.Stat(serviceAccountTokenInClusterPath); err == nil {
			return "", true, nil
		}
	}

	if filePath := os.Getenv("KUBECONFIG"); filePath != "" {
		return filePath, false, nil
	}

	homedirPath := homedir.HomeDir()
	if homedirPath == "" {
		return "", false, fmt.Errorf("failed to determine homedir path")
	}

	return filepath.Join(homedirPath, ".kube", "config"), false, nil
}

func LoadKubeconfig(configFilePath string) (KubeConfig, error) {
	log.Infof("Using kubeconfig from '%v'", configFilePath)

	kubeconfig, err := clientcmd.LoadFromFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from '%v': %v", configFilePath, err)
	}

	return kubeconfig, nil
}

func ContextNames(
	kubeconfig KubeConfig,
	selectedName string,
	allContexts bool,
	excludedNames []string,
) ([]string, error) {
	if !allContexts && selectedName == "" {
		currentContext := kubeconfig.CurrentContext
		log.Infof("No context name provided, will use current context: %v", currentContext)
		return []string{currentContext}, nil

	}
	namesSet := contextNames(kubeconfig)
	if !allContexts {
		_, found := namesSet[selectedName]
		if found {
			log.Infof("Will use context %v", selectedName)
			return []string{selectedName}, nil
		}
		return nil, fmt.Errorf("selected context '%v' is not found in kubeconfig", selectedName)
	}

	for _, name := range excludedNames {
		delete(namesSet, name)
	}

	log.Infof("Will iterate %v contexts", len(namesSet))

	names := internal.Keys(namesSet)
	sort.Strings(names)
	return names, nil
}

func contextNames(kubeconfig KubeConfig) map[string]bool {
	var names []string
	for name := range kubeconfig.Contexts {
		if name != "" {
			names = append(names, name)
		}
	}
	return internal.ToBoolMap(names)
}
