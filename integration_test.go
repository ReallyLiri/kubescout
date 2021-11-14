// +build integration

package main

import (
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/internal/kubeclient"
	"github.com/reallyliri/kubescout/internal/kubeconfig"
	"github.com/reallyliri/kubescout/pkg"
	"github.com/reallyliri/kubescout/sink"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

var integrationTestResourcesFilePath string

func init() {
	_, filePath, _, _ := runtime.Caller(0)
	integrationTestResourcesFilePath = path.Join(filePath, "../test-resources/integration-test-resources.yaml")
}

func runCommand(executable string, args ...string) (string, error) {
	cmd := exec.Command(executable, args...)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err != nil {
		return "", fmt.Errorf("command failed: %v\n%v", err, outputStr)
	}
	return outputStr, nil
}

func verifyMinikubeRunning() error {
	output, err := runCommand("minikube", "status")
	if err != nil {
		return err
	}
	if strings.Contains(output, "Stopped") || !strings.Contains(output, "Running") {
		return fmt.Errorf("minikube is not running, please set it up and re-run the test")
	}
	return nil
}

func verifyKubeconfigSetToMinikube() error {
	output, err := runCommand("kubectl", "config", "current-context")
	if err != nil {
		return err
	}
	if output != "minikube" {
		return fmt.Errorf("current kube context is set to '%v', please set it to minikube and re-run the test", output)
	}
	return nil
}

func applyManifests() error {
	_, err := runCommand("kubectl", "-n", "default", "apply", "-f", integrationTestResourcesFilePath)
	if err != nil {
		return fmt.Errorf("failed to apply manifests from %v: %v", integrationTestResourcesFilePath, err)
	}
	return nil
}

func cleanupDefaultNamespace() error {
	_, err := runCommand("kubectl", "-n", "default", "delete", "deploy", "--all", "--timeout=30s")
	if err != nil {
		return fmt.Errorf("failed to clear deployments from %v: %v", integrationTestResourcesFilePath, err)
	}
	_, err = runCommand("kubectl", "-n", "default", "delete", "event", "--all", "--timeout=30s")
	if err != nil {
		return fmt.Errorf("failed to clear events from %v: %v", integrationTestResourcesFilePath, err)
	}
	return nil
}

func verifyClusterInitialState(t *testing.T, client kubeclient.KubernetesClient) {
	namespaces, err := client.GetNamespaces()
	require.Nil(t, err)
	require.Equal(t, 4, len(namespaces))
	require.Equal(t, "default", namespaces[0].Name)
	require.Equal(t, "kube-node-lease", namespaces[1].Name)
	require.Equal(t, "kube-public", namespaces[2].Name)
	require.Equal(t, "kube-system", namespaces[3].Name)

	pods, err := client.GetPods("default")
	require.Nil(t, err)
	require.Equal(t, 0, len(pods))
}

func verifyClusterReadyForTest(t *testing.T, client kubeclient.KubernetesClient) {
	namespaces, err := client.GetNamespaces()
	require.Nil(t, err)
	require.Equal(t, 4, len(namespaces))
	require.Equal(t, "default", namespaces[0].Name)
	require.Equal(t, "kube-node-lease", namespaces[1].Name)
	require.Equal(t, "kube-public", namespaces[2].Name)
	require.Equal(t, "kube-system", namespaces[3].Name)

	pods, err := client.GetPods("default")
	require.Nil(t, err)
	require.Equal(t, 6, len(pods))
}

type verifySink struct {
	alerts *alert.Alerts
}

var _ sink.Sink = &verifySink{}

func (s *verifySink) Report(alerts *alert.Alerts) error {
	s.alerts = alerts
	return nil
}

func cleanUp() {
	log.Infof("cleaning up namespace ...\n")
	err := cleanupDefaultNamespace()
	if err != nil {
		log.Error(err.Error() + "\n")
	}
}

func TestIntegration(t *testing.T) {
	cfg, err := config.DefaultConfig()
	require.Nil(t, err)

	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	log.Infof("using store file at '%v'\n", storeFile.Name())
	cfg.StoreFilePath = storeFile.Name()
	cfg.MessagesDeduplicationDuration = time.Minute
	cfg.IncludeNamespaces = []string{"default"}
	cfg.OutputMode = "discard"
	cfg.ContextName = "minikube"

	err = verifyMinikubeRunning()
	require.Nil(t, err)

	err = verifyKubeconfigSetToMinikube()
	require.Nil(t, err)

	kconf, err := kubeconfig.LoadKubeconfig(cfg.KubeconfigFilePath)
	require.Nil(t, err)
	client, err := kubeclient.CreateClient(cfg, kconf)
	require.Nil(t, err)

	verifyClusterInitialState(t, client)

	err = applyManifests()
	require.Nil(t, err)

	log.Infof("set up completed, sleeping to give namespace time to stabilize ...\n")

	defer cleanUp()

	time.Sleep(time.Minute * time.Duration(3))

	verifyClusterReadyForTest(t, client)

	vSink := &verifySink{}

	log.Infof("running 1/3 diagnose call ...\n")
	err = pkg.Scout(cfg, vSink)
	require.Nil(t, err)

	require.NotNil(t, vSink.alerts)
	assert.Equal(t, 1, len(vSink.alerts.AlertsByClusterName))
	assert.NotNil(t, vSink.alerts.AlertsByClusterName["minikube"])
	verifyAlertsForFullDiagRun(t, vSink.alerts.AlertsByClusterName["minikube"])
	vSink.alerts = nil

	storeContent, err := ioutil.ReadFile(cfg.StoreFilePath)
	require.Nil(t, err)
	require.NotEmpty(t, storeContent)

	log.Infof("running 2/3 diagnose call ...\n")
	err = pkg.Scout(cfg, vSink)
	require.Nil(t, err)

	assert.Nil(t, vSink.alerts)
	if vSink.alerts != nil {
		for _, entityAlert := range vSink.alerts.AlertsByClusterName["minikube"] {
			assert.Fail(t, entityAlert.String())
		}
		vSink.alerts = nil
	}

	log.Infof("sleeping to get de-dup grace time to pass")
	time.Sleep(time.Minute)

	log.Infof("running 3/3 diagnose call ...\n")
	err = pkg.Scout(cfg, vSink)
	require.Nil(t, err)

	require.NotNil(t, vSink.alerts)
	assert.Equal(t, 1, len(vSink.alerts.AlertsByClusterName))
	assert.NotNil(t, vSink.alerts.AlertsByClusterName["minikube"])
	verifyAlertsForFullDiagRun(t, vSink.alerts.AlertsByClusterName["minikube"])
	vSink.alerts = nil
}

func assertMessage(t *testing.T, expectedFormat string, actualMessage string) {
	expectedFormatRegex := "(?s)" + strings.ReplaceAll(regexp.QuoteMeta(expectedFormat), "\\*", ".*")
	matched, err := regexp.MatchString(expectedFormatRegex, actualMessage)
	require.Nil(t, err)
	assert.True(t, matched, "did not match\nExpected:\n%v\nActual:\n%v", expectedFormat, actualMessage)
}

func verifyAlertsForFullDiagRun(t *testing.T, alerts alert.EntityAlerts) {
	assert.Equal(t, 5, len(alerts))
	for i, entityAlert := range alerts {
		assertMessage(t, expectedFormatsFirstRun[i], entityAlert.String())
	}
}

var expectedFormatsFirstRun = []string{
	`Pod default/test-2-broken-image-* is un-healthy:
Container test-2-broken-image still waiting due to *
Event by kubelet: Failed x* since * (last seen * ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown
Event by kubelet: Failed x* since * (last seen * ago):
	Error: ErrImagePull`,

	`Pod default/test-3-excessive-resources-* is un-healthy:
Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: * ago)
Event by default-scheduler: FailedScheduling *since * ago*:
	0/1 nodes are available: 1 Insufficient memory.`,

	`Pod default/test-4-crashlooping-* is un-healthy:
*test-4-crashlooping *
Event by kubelet: BackOff x* since * (last seen * ago):
	Back-off restarting failed container
Logs of container test-4-crashlooping:
--------
1
2
3
4
5
--------`,

	`Pod default/test-5-completed-* is un-healthy:
*test-5-completed *
Event by kubelet: BackOff x* since * (last seen * ago):
	Back-off restarting failed container
Logs of container test-5-completed:
--------
1
2
3
4
5
--------`,

	`Pod default/test-6-crashlooping-init-* is un-healthy:
*test-6-crashlooping-init-container (init) *
Event by kubelet: BackOff x* since * (last seen * ago):
	Back-off restarting failed container
Logs of container test-6-crashlooping-init-container:
--------
1
2
3
4
5
--------`,

	`unexpected`,
}
