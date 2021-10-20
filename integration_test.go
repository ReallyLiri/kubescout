// +build integration

package main

import (
	"KubeScout/config"
	"KubeScout/diag"
	"KubeScout/kubeclient"
	"KubeScout/store"
	"fmt"
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

func TestIntegration(t *testing.T) {
	configuration, err := config.DefaultConfig()
	require.Nil(t, err)

	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	log.Infof("using store file at '%v'\n", storeFile.Name())
	configuration.ClusterName = "integration-test"
	configuration.StoreFilePath = storeFile.Name()
	configuration.MessagesDeduplicationDuration = time.Minute
	configuration.ExcludeNamespaces = []string{"kube-system"}

	err = verifyMinikubeRunning()
	require.Nil(t, err)

	err = verifyKubeconfigSetToMinikube()
	require.Nil(t, err)

	client, err := kubeclient.CreateClient(configuration)
	require.Nil(t, err)

	verifyClusterInitialState(t, client)

	err = applyManifests()
	require.Nil(t, err)

	log.Infof("applied manifests, sleeping to give namespace time to stabilize ...\n")

	defer func() {
		log.Infof("cleaning up namespace ...\n")
		err := cleanupDefaultNamespace()
		if err != nil {
			log.Infof(err.Error())
		}
	}()

	time.Sleep(time.Minute * time.Duration(2))

	verifyClusterReadyForTest(t, client)

	storeForFirstRun, err := store.LoadOrCreate(configuration)
	require.Nil(t, err)

	log.Infof("running 1/3 diagnose call ...\n")
	err = diag.DiagnoseCluster(client, configuration, storeForFirstRun, time.Now().UTC())
	require.Nil(t, err)

	relevantMessagesFirstRun := storeForFirstRun.RelevantMessages()
	verifyMessagesForFullDiagRun(t, relevantMessagesFirstRun)

	storeContent, err := ioutil.ReadFile(configuration.StoreFilePath)
	require.Nil(t, err)
	require.NotEmpty(t, storeContent)

	storeForSecondRun, err := store.LoadOrCreate(configuration)
	require.Nil(t, err)

	log.Infof("running 2/3 diagnose call ...\n")
	err = diag.DiagnoseCluster(client, configuration, storeForSecondRun, time.Now().UTC())
	require.Nil(t, err)

	relevantMessagesSecondRun := storeForSecondRun.RelevantMessages()
	verifyMessagesForSilencedRun(t, relevantMessagesSecondRun)

	log.Infof("sleeping to get de-dup grace time to pass")
	time.Sleep(time.Minute)

	storeForThirdRun, err := store.LoadOrCreate(configuration)
	require.Nil(t, err)

	log.Infof("running 3/3 diagnose call ...\n")
	err = diag.DiagnoseCluster(client, configuration, storeForThirdRun, time.Now().UTC())
	require.Nil(t, err)

	relevantMessagesThirdRun := storeForThirdRun.RelevantMessages()
	verifyMessagesForFullDiagRun(t, relevantMessagesThirdRun)
}

func assertMessage(t *testing.T, expectedFormat string, actualMessage string) {
	expectedFormatRegex := "(?s)" + strings.ReplaceAll(regexp.QuoteMeta(expectedFormat), "\\*", ".*")
	matched, err := regexp.MatchString(expectedFormatRegex, actualMessage)
	require.Nil(t, err)
	assert.True(t, matched, "did not match\nExpected:\n%v\nActual:\n%v", expectedFormat, actualMessage)
}

func verifyMessagesForFullDiagRun(t *testing.T, messages []string) {
	assert.Equal(t, 12, len(messages))
	for i, message := range messages {
		assertMessage(t, expectedFormatsFirstRun[i], message)
	}
}

func verifyMessagesForSilencedRun(t *testing.T, messages []string) {
	// ideally we'd have 0 messages, but sometimes we get some first run messages on delay and they shouldn't be silenced
	if len(messages) > 3 {
		assert.Fail(t, "too many messages on second run: %v", len(messages))
	}
	expectedFormat := `Pod default/test-* is un-healthy
	test-*
logs of container test-*:
<<<<<<<<<<
1
2
3
4
5

>>>>>>>>>>`
	for _, message := range messages {
		assertMessage(t, expectedFormat, message)
	}
}

var expectedFormatsFirstRun = []string{
	`Pod default/test-2-broken-image-* is un-healthy
	Pod is in Pending phase
	test-2-broken-image still waiting due to *`,

	`Event on Pod test-2-broken-image-* due to Failed (at *, * ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`,

	`Event on Pod test-2-broken-image-* due to Failed (at *, * ago):
	Error: ErrImagePull`,

	`Event on Pod test-2-broken-image-* due to Failed (at *, *):
	Error: ImagePullBackOff`,

	`Pod default/test-3-excessive-resources-* is un-healthy
	Pod is in Pending phase
	Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: * ago)`,

	`Event on Pod test-3-excessive-resources-* due to FailedScheduling (at *, * ago):
	0/1 nodes are available: 1 Insufficient memory.`,

	`Pod default/test-4-crashlooping-* is un-healthy
*	test-4-crashlooping had restarted * times, last exit due to Error (exit code 1)*
logs of container test-4-crashlooping:
<<<<<<<<<<
1
2
3
4
5

>>>>>>>>>>`,

	`Event on Pod test-4-crashlooping-* due to BackOff (at *, * ago):
	Back-off restarting failed container`,

	`Pod default/test-5-* is un-healthy
*	test-5-completed had restarted * times, last exit due to Completed (exit code 0)*
logs of container test-5-completed:
<<<<<<<<<<
1
2
3
4
5

>>>>>>>>>>`,

	`Event on Pod test-5-completed-* due to BackOff (at *, * ago):
	Back-off restarting failed container`,

	`Pod default/test-6-crashlooping-init-* is un-healthy
	Pod is in Pending phase
*	test-6-crashlooping-init-container (init) *
logs of container test-6-crashlooping-init-container:
<<<<<<<<<<
1
2
3
4
5

>>>>>>>>>>`,

	`Event on Pod test-6-crashlooping-init-* due to BackOff (at *, * ago):
	Back-off restarting failed container`,
}
