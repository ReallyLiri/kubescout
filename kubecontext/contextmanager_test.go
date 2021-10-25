package kubecontext

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
	manager, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "", manager.GetCurrentContext())
	names, err := manager.GetContextNames()
	require.Nil(t, err)
	assert.Empty(t, names)
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
	manager, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "", manager.GetCurrentContext())
	names, err := manager.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)

	err = manager.SetCurrentContext("cluster1")
	require.Nil(t, err)

	assert.Equal(t, "cluster1", manager.GetCurrentContext())
	names, err = manager.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)

	err = manager.TearDown()
	require.Nil(t, err)

	manager2, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)

	assert.Equal(t, "", manager2.GetCurrentContext())
	names, err = manager2.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)
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
	manager, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "cluster2", manager.GetCurrentContext())
	names, err := manager.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)

	err = manager.SetCurrentContext("cluster1")
	require.Nil(t, err)

	assert.Equal(t, "cluster1", manager.GetCurrentContext())
	names, err = manager.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)

	err = manager.TearDown()
	require.Nil(t, err)

	manager2, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)

	assert.Equal(t, "cluster2", manager2.GetCurrentContext())
	names, err = manager2.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)
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
	manager, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "", manager.GetCurrentContext())
	names, err := manager.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)

	err = manager.SetCurrentContext("cluster1")
	require.Nil(t, err)

	assert.Equal(t, "cluster1", manager.GetCurrentContext())
	names, err = manager.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)

	err = manager.TearDown()
	require.Nil(t, err)

	manager2, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)

	assert.Equal(t, "", manager2.GetCurrentContext())
	names, err = manager2.GetContextNames()
	require.Nil(t, err)
	assert.Equal(t, []string{"cluster1", "cluster2"}, names)
}
