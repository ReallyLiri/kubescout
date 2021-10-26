package kubeconfig

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func createKubeconfig(t *testing.T, content string) string {
	kubeconfigPath := filepath.Join(t.TempDir(), "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(content), 0777)
	require.Nil(t, err)
	return kubeconfigPath
}

func Test_LoadEmptyContextsFile(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
contexts:
- context:
current-context: ""
kind: Config
preferences: {}
users:
- name: user1
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "", kubeconfig.CurrentContext)
}

func Test_LoadConfigWithContexts(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
contexts:
- context:
    cluster: cluster1
    user: user1
  name: cluster1
- context:
    cluster: cluster1
    user: user2
  name: cluster2
current-context: ""
kind: Config
preferences: {}
users:
- name: user1
  user: {}
- name: user2
  user: {}

`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "", kubeconfig.CurrentContext)
}

func Test_LoadConfigWithContextsAndNoEmptyCurrent(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
contexts:
- context:
    cluster: cluster1
    user: user1
  name: cluster1
- context:
    cluster: cluster1
    user: user2
  name: cluster2
current-context: "cluster2"
kind: Config
preferences: {}
users:
- name: user1
  user: {}
- name: user2
  user: {}

`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "cluster2", kubeconfig.CurrentContext)
}

func Test_LoadConfigWithContextsAndNoCurrentContextNode(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
contexts:
- context:
    cluster: cluster1
    user: user1
  name: cluster1
- context:
    cluster: cluster1
    user: user2
  name: cluster2
kind: Config
preferences: {}
users:
- name: user1
  user: {}
- name: user2
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "", kubeconfig.CurrentContext)
}

func TestContextNames_WithSelectedContextConfig(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
- cluster:
    server: ""
  name: cluster2
- cluster:
    server: ""
  name: cluster3
contexts:
- context:
    cluster: cluster1
    user: user
  name: c1
- context:
    cluster: cluster2
    user: user
  name: c2
- context:
    cluster: cluster3
    user: user
  name: c3
kind: Config
preferences: {}
users:
- name: user
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)

	names, err := ContextNames(
		kubeconfig,
		"c1",
		false,
		[]string{},
	)

	require.Nil(t, err)
	require.Equal(t, []string{"c1"}, names)
}

func TestContextNames_WithSelectedContextConfigThatDoesntMatchCurrent(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
- cluster:
    server: ""
  name: cluster2
- cluster:
    server: ""
  name: cluster3
contexts:
- context:
    cluster: cluster1
    user: user
  name: c1
- context:
    cluster: cluster2
    user: user
  name: c2
- context:
    cluster: cluster3
    user: user
  name: c3
kind: Config
preferences: {}
users:
- name: user
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)

	names, err := ContextNames(
		kubeconfig,
		"c2",
		false,
		[]string{},
	)

	require.Nil(t, err)
	require.Equal(t, []string{"c2"}, names)
}

func TestContextNames_WithSelectedContextConfigThatDoesntExist(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
- cluster:
    server: ""
  name: cluster2
- cluster:
    server: ""
  name: cluster3
contexts:
- context:
    cluster: cluster1
    user: user
  name: c1
- context:
    cluster: cluster2
    user: user
  name: c2
- context:
    cluster: cluster3
    user: user
  name: c3
kind: Config
preferences: {}
users:
- name: user
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)

	_, err = ContextNames(
		kubeconfig,
		"c7",
		false,
		[]string{},
	)

	require.NotNil(t, err)
}

func TestContextNames_WithAllContextsSelected(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
- cluster:
    server: ""
  name: cluster2
- cluster:
    server: ""
  name: cluster3
contexts:
- context:
    cluster: cluster1
    user: user
  name: c1
- context:
    cluster: cluster2
    user: user
  name: c2
- context:
    cluster: cluster3
    user: user
  name: c3
kind: Config
preferences: {}
users:
- name: user
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)

	names, err := ContextNames(
		kubeconfig,
		"",
		true,
		[]string{},
	)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c2", "c3"}, names)
}

func TestContextNames_WithSpecificContextAndAllContextsSelected(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
- cluster:
    server: ""
  name: cluster2
- cluster:
    server: ""
  name: cluster3
contexts:
- context:
    cluster: cluster1
    user: user
  name: c1
- context:
    cluster: cluster2
    user: user
  name: c2
- context:
    cluster: cluster3
    user: user
  name: c3
kind: Config
preferences: {}
users:
- name: user
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)

	names, err := ContextNames(
		kubeconfig,
		"c7",
		true,
		[]string{},
	)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c2", "c3"}, names)
}

func TestContextNames_WithAllContextsSelectedAndOneExcluded(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
- cluster:
    server: ""
  name: cluster2
- cluster:
    server: ""
  name: cluster3
contexts:
- context:
    cluster: cluster1
    user: user
  name: c1
- context:
    cluster: cluster2
    user: user
  name: c2
- context:
    cluster: cluster3
    user: user
  name: c3
kind: Config
preferences: {}
users:
- name: user
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)

	names, err := ContextNames(
		kubeconfig,
		"",
		true,
		[]string{"c2"},
	)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c3"}, names)
}

func TestContextNames_WithAllContextsSelectedAndSomeExcluded(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    server: ""
  name: cluster1
- cluster:
    server: ""
  name: cluster2
- cluster:
    server: ""
  name: cluster3
- cluster:
    server: ""
  name: cluster4
contexts:
- context:
    cluster: cluster1
    user: user
  name: c1
- context:
    cluster: cluster2
    user: user
  name: c2
- context:
    cluster: cluster3
    user: user
  name: c3
- context:
    cluster: cluster4
    user: user
  name: c4
kind: Config
preferences: {}
users:
- name: user
  user: {}
`
	kubeconfigPath := createKubeconfig(t, content)
	kubeconfig, err := LoadKubeconfig(kubeconfigPath)

	names, err := ContextNames(
		kubeconfig,
		"",
		true,
		[]string{"c3", "c2"},
	)

	require.Nil(t, err)
	require.Equal(t, []string{"c1", "c4"}, names)
}
