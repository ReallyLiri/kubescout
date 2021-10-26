package config

import (
	"fmt"
	"github.com/reallyliri/kubescout/internal"
	"github.com/reallyliri/kubescout/kubecontext"
	log "github.com/sirupsen/logrus"
	"sort"
)

func (config *Config) ContextNames(contextManager kubecontext.ConfigContextManager) ([]string, error) {
	nameToUse := config.ContextName
	if !config.AllContexts && nameToUse == "" {
		currentContext := contextManager.GetCurrentContext()
		log.Infof("No context name provided, will use current context: %v", currentContext)
		return []string{currentContext}, nil

	}
	names, err := contextManager.GetContextNames()
	if err != nil {
		return nil, err
	}
	namesSet := internal.ToBoolMap(names)
	if !config.AllContexts {
		_, found := namesSet[nameToUse]
		if found {
			log.Infof("Will use context %v", nameToUse)
			return []string{nameToUse}, nil
		}
		return nil, fmt.Errorf("selected context '%v' is not found in kubeconfig", nameToUse)
	}

	for _, name := range config.ExcludeContexts {
		delete(namesSet, name)
	}

	log.Infof("Will iterate %v contexts", len(namesSet))

	names = internal.Keys(namesSet)
	sort.Strings(names)
	return names, nil
}
