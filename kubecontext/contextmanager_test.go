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

func Test_LoadComplicatedContext(t *testing.T) {
	// language=yaml
	content := `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: arn:aws:eks:us-east-1:496712460628:cluster/test-nfs
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: arn:aws:eks:us-east-2:975728162598:cluster/test
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-b_liri-test
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-c_app-cluster
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-c_liri-dev
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-c_liri-interview
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-c_liri-test
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1_app-svc
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1_app-services
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-east1-b_fun-time
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-east1-b_liri-cluster
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app-velo_europe-west2_velo
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-b_app-cluster
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-b_app-services
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-c_app-kube-fun
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1-c_app-test
- cluster:
    certificate-authority-data: LS0tLS1
    server: https:/1.1.1.1
  name: gke_app_us-central1_app-svc
- cluster:
    certificate-authority: /Users/reallyliri/.minikube/ca.crt
    extensions:
    - extension:
        last-update: Tue, 19 Oct 2021 16:10:17 IDT
        provider: minikube.sigs.k8s.io
        version: v1.23.2
      name: cluster_info
    server: https:/1.1.1.1
  name: minikube
contexts:
- context:
    cluster: gke_app_us-central1_app-svc
    namespace: defaualt
    user: gke_app_us-central1_app-svc
  name: gke_app_us-central1_app-svc
- context:
    cluster: gke_app_us-central1_app-services
    user: gke_app_us-central1_app-services
  name: gke_app_us-central1_app-services
- context:
    cluster: gke_app_us-east1-b_liri-cluster
    user: gke_app_us-east1-b_liri-cluster
  name: gke_app_us-east1-b_liri-cluster
- context:
    cluster: gke_app_us-central1-b_app-cluster
    user: gke_app_us-central1-b_app-cluster
  name: gke_app_us-central1-b_app-cluster
- context:
    cluster: gke_app_us-central1-b_app-services
    user: gke_app_us-central1-b_app-services
  name: gke_app_us-central1-b_app-services
- context:
    cluster: gke_app_us-central1_app-svc
    user: gke_app_us-central1_app-svc
  name: gke_app_us-central1_app-svc
- context:
    cluster: gke_app_us-central1-b_app-cluster
    namespace: smoke-test
    user: gke_app_us-central1-b_app-cluster
  name: api
- context:
    cluster: minikube
    extensions:
    - extension:
        last-update: Tue, 19 Oct 2021 16:10:17 IDT
        provider: minikube.sigs.k8s.io
        version: v1.23.2
      name: context_info
    namespace: default
    user: minikube
  name: minikube
- context:
    cluster: gke_app_us-central1-c_app-cluster
    namespace: smoke-test
    user: gke_app_us-central1-c_app-cluster
  name: rnd
- context:
    cluster: gke_app_us-central1_app-svc
    namespace: gke-app-self-serv-svc-poo-e7c093e6-k2ee
    user: gke_app_us-central1_app-svc
  name: sese
- context:
    cluster: gke_app_us-central1-b_app-services
    namespace: default
    user: gke_app_us-central1-b_app-services
  name: svc
- context:
    cluster: gke_app-velo_europe-west2_velo
    namespace: liri-softrev
    user: gke_app-velo_europe-west2_velo
  name: velo
current-context: rnd
kind: Config
preferences: {}
users:
- name: arn:aws:eks:us-east-1:496712460628:cluster/test-nfs
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - --region
      - us-east-1
      - eks
      - get-token
      - --cluster-name
      - test-nfs
      command: aws
      env: null
      provideClusterInfo: false
- name: arn:aws:eks:us-east-2:975728162598:cluster/test
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - --region
      - us-east-2
      - eks
      - get-token
      - --cluster-name
      - test
      command: aws
      env: null
      provideClusterInfo: false
- name: gke_app_us-central1-b_liri-test
  user:
    auth-provider:
      config:
        access-token: ya29.a0A
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-06-29T15:11:22Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-c_app-cluster
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-25T10:48:24Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-c_liri-dev
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-06-02T08:26:54Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-c_liri-interview
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-09-09T06:34:25Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-c_liri-test
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-05-13T11:29:31Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1_app-svc
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-23T15:44:16Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1_app-services
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-23T15:44:16Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-east1-b_fun-time
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-12T15:14:18Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-east1-b_liri-cluster
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-08-18T12:42:56Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app-velo_europe-west2_velo
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-10T12:58:46Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-b_app-cluster
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-25T10:48:24Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-b_app-services
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-23T15:44:16Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-c_app-kube-fun
  user:
    auth-provider:
      config:
        access-token: ya29.A0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-02-25T10:00:34Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1-c_app-test
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2020-11-24T09:41:26Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: gke_app_us-central1_app-svc
  user:
    auth-provider:
      config:
        access-token: ya29.a0
        cmd-args: config config-helper --format=json
        cmd-path: /Users/reallyliri/google-cloud-sdk/bin/gcloud
        expiry: "2021-10-23T15:44:16Z"
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp
- name: minikube
  user:
    client-certificate: /Users/reallyliri/.minikube/profiles/minikube/client.crt
    client-key: /Users/reallyliri/.minikube/profiles/minikube/client.key
- name: velo@VELO2SB5
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - auth
      - cluster
      command: /usr/local/bin/velo
      env: null
      provideClusterInfo: false

`
	kubeconfigPath := createKubeconfig(t, content)
	manager, err := LoadKubeConfig(kubeconfigPath)
	require.Nil(t, err)
	assert.Equal(t, "rnd", manager.GetCurrentContext())
	names, err := manager.GetContextNames()
	require.Nil(t, err)
	expectedNames := []string{
		"api",
		"gke_app_us-central1-b_app-cluster", "gke_app_us-central1-b_app-services", "gke_app_us-central1_app-services",
		"gke_app_us-central1_app-svc", "gke_app_us-central1_app-svc", "gke_app_us-east1-b_liri-cluster",
		"minikube", "rnd", "sese", "svc", "velo"}
	assert.Equal(t, expectedNames, names)
}
