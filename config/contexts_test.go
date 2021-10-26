package config

import (
	"github.com/reallyliri/kubescout/kubecontext"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContextNames_WithSelectedContextConfig(t *testing.T) {
	manager := kubecontext.CreateConfigContextManagerMock([]string{"c3", "c2", "c1"}, "c1")

	config := &Config{
		ContextName:     "c1",
		AllContexts:     false,
		ExcludeContexts: []string{},
	}

	names, err := config.ContextNames(manager)

	require.Nil(t, err)
	require.Equal(t, []string{"c1"}, names)
}

func TestContextNames_WithSelectedContextConfigThatDoesntMatchCurrent(t *testing.T) {
	manager := kubecontext.CreateConfigContextManagerMock([]string{"c3", "c2", "c1"}, "c1")

	config := &Config{
		ContextName:     "c2",
		AllContexts:     false,
		ExcludeContexts: []string{},
	}

	names, err := config.ContextNames(manager)

	require.Nil(t, err)
	require.Equal(t, []string{"c2"}, names)
}

func TestContextNames_WithSelectedContextConfigThatDoesntExist(t *testing.T) {
	manager := kubecontext.CreateConfigContextManagerMock([]string{"c3", "c2", "c1"}, "c1")

	config := &Config{
		ContextName:     "c7",
		AllContexts:     false,
		ExcludeContexts: []string{},
	}

	_, err := config.ContextNames(manager)

	require.NotNil(t, err)
}

func TestContextNames_WithAllContextsSelected(t *testing.T) {
	manager := kubecontext.CreateConfigContextManagerMock([]string{"c3", "c2", "c1"}, "c1")

	config := &Config{
		ContextName:     "",
		AllContexts:     true,
		ExcludeContexts: []string{},
	}

	names, err := config.ContextNames(manager)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c2", "c3"}, names)
}

func TestContextNames_WithSpecificContextAndAllContextsSelected(t *testing.T) {
	manager := kubecontext.CreateConfigContextManagerMock([]string{"c3", "c2", "c1"}, "c1")

	config := &Config{
		ContextName:     "c7",
		AllContexts:     true,
		ExcludeContexts: []string{},
	}

	names, err := config.ContextNames(manager)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c2", "c3"}, names)
}

func TestContextNames_WithAllContextsSelectedAndOneExcluded(t *testing.T) {
	manager := kubecontext.CreateConfigContextManagerMock([]string{"c3", "c2", "c1"}, "c1")

	config := &Config{
		ContextName:     "",
		AllContexts:     true,
		ExcludeContexts: []string{"c2"},
	}

	names, err := config.ContextNames(manager)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c3"}, names)
}

func TestContextNames_WithAllContextsSelectedAndSomeExcluded(t *testing.T) {
	manager := kubecontext.CreateConfigContextManagerMock([]string{"c3", "c2", "c1", "c4"}, "c1")

	config := &Config{
		ContextName:     "",
		AllContexts:     true,
		ExcludeContexts: []string{"c3", "c2"},
	}

	names, err := config.ContextNames(manager)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c4"}, names)
}
