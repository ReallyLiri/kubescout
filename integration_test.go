package main

import (
	"KubeScout/config"
	"KubeScout/diag"
	"KubeScout/kubeclient"
	"KubeScout/store"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"log"
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

func createStore() {

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
	flagsSet, err := config.FlagSet("integration-test")
	require.Nil(t, err)

	cfg, err := config.ParseConfig(cli.NewContext(nil, flagsSet, nil))
	require.Nil(t, err)

	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	log.Printf("using store file at '%v'\n", storeFile.Name())
	cfg.StoreFilePath = storeFile.Name()
	cfg.MessagesDeduplicationDuration = time.Minute

	err = verifyMinikubeRunning()
	require.Nil(t, err)

	err = verifyKubeconfigSetToMinikube()
	require.Nil(t, err)

	client, err := kubeclient.CreateClient(cfg)
	require.Nil(t, err)

	verifyClusterInitialState(t, client)

	err = applyManifests()
	require.Nil(t, err)

	log.Printf("applied manifests, sleeping to give namespace time to stabilize ...\n")

	defer func() {
		log.Printf("cleaning up namespace ...\n")
		err := cleanupDefaultNamespace()
		if err != nil {
			log.Printf(err.Error())
		}
	}()

	time.Sleep(time.Minute * time.Duration(2))

	verifyClusterReadyForTest(t, client)

	storeForFirstRun, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	log.Printf("running 1/3 diagnose call ...\n")
	err = diag.DiagnoseCluster(client, cfg, storeForFirstRun, time.Now().UTC())
	require.Nil(t, err)

	relevantMessagesFirstRun := storeForFirstRun.RelevantMessages()
	verifyMessages(t, relevantMessagesFirstRun)

	storeContent, err := ioutil.ReadFile(cfg.StoreFilePath)
	require.Nil(t, err)
	require.NotEmpty(t, storeContent)

	storeForSecondRun, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	log.Printf("running 2/3 diagnose call ...\n")
	err = diag.DiagnoseCluster(client, cfg, storeForSecondRun, time.Now().UTC())
	require.Nil(t, err)

	relevantMessagesSecondRun := storeForFirstRun.RelevantMessages()
	assert.Equal(t, 0, len(relevantMessagesSecondRun))

	log.Printf("sleeping to get de-dup grace time to pass")
	time.Sleep(time.Minute)

	storeForThirdRun, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	log.Printf("running 3/3 diagnose call ...\n")
	err = diag.DiagnoseCluster(client, cfg, storeForThirdRun, time.Now().UTC())
	require.Nil(t, err)

	relevantMessagesThirdRun := storeForFirstRun.RelevantMessages()
	verifyMessages(t, relevantMessagesThirdRun)
}

var expectedMessages = []string{
	`Pod default/test-2-broken-image-XXX is un-healthy
	Pod is in Pending phase
	test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image "nginx:l4t3st"`,

	`Pod default/test-3-excessive-resources-XXX is un-healthy
	Pod is in Pending phase
	Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 2 minutes ago)`,

	`Pod default/test-4-crashlooping-XXX is un-healthy
	test-4-crashlooping still waiting due to CrashLoopBackOff: back-off ZZs restarting failed container
	test-4-crashlooping had restarted 4 times last exit due to Error (exit code 1)
logs of container test-4-crashlooping:
<<<<<<<<<<
1
2
3
4
5
>>>>>>>>>>`,

	`Pod default/test-5-completed-XXX is un-healthy
	test-5-completed still waiting due to CrashLoopBackOff: back-off ZZs restarting failed container
	test-5-completed had restarted 4 times last exit due to Completed (exit code 0)
logs of container test-5-completed:
<<<<<<<<<<
1
2
3
4
5
>>>>>>>>>>`,

	`Pod default/test-6-crashlooping-init-XXX is un-healthy
	Pod is in Pending phase
	test-6-crashlooping-init-container (init) still waiting due to CrashLoopBackOff: back-off ZZs restarting failed container
logs of container test-6-crashlooping-init-container:
<<<<<<<<<<
1
2
3
4
5
>>>>>>>>>>`,

	`Event default/test-2-broken-image-XXX.YYY is un-healthy
	Event on Pod test-2-broken-image-XXX due to Failed (at some time ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`,

	`Event default/test-2-broken-image-XXX.YYY is un-healthy
	Event on Pod test-2-broken-image-XXX due to Failed (at some time ago):
	Error: ErrImagePull`,

	`Event default/test-2-broken-image-XXX.YYY is un-healthy
	Event on Pod test-2-broken-image-XXX due to Failed (at some time ago):
	Error: ImagePullBackOff`,

	`Event default/test-3-excessive-resources-XXX.YYY is un-healthy
	Event on Pod test-3-excessive-resources-XXX due to FailedScheduling (at some time ago):
	0/1 nodes are available: 1 Insufficient memory.`,

	`Event default/test-3-excessive-resources-XXX.YYY ZZs un-healthy
	Event on Pod test-3-excessive-resources-XXX to FailedScheduling (at unavailable time, unknown time ago):
	0/1 nodes are available: 1 Insufficient memory.`,

	`Event default/test-4-crashlooping-XXX.YYY is un-healthy
	Event on Pod test-4-crashlooping-XXX due to BackOff (at some time ago):
	Back-off restarting failed container`,

	`Event default/test-5-completed-XXX.YYY is un-healthy
	Event on Pod test-5-completed-XXX due to BackOff (at some time ago):
	Back-off restarting failed container`,

	`Event default/test-6-crashlooping-init-XXX.YYY is un-healthy
	Event on Pod test-6-crashlooping-init-XXX due to BackOff (at some time ago):
	Back-off restarting failed container`,
}

func verifyMessages(t *testing.T, messages []string) {
	assert.Equal(t, 13, len(messages))

	podNameSuffixRegex, err := regexp.Compile(`-(?:.{9}|.{10})-.{5} `)
	require.Nil(t, err)
	eventNameSuffixRegex, err := regexp.Compile(`-(?:.{9}|.{10})-.{5}\..{16}`)
	require.Nil(t, err)
	atBlockRegex, err := regexp.Compile(`\(at (?:\d{4}|\d{2}) .* (?:\d{4}|\d{2}) (?:\d{4}|\d{2}):(?:\d{4}|\d{2}) .*, .* ago\)`)
	require.Nil(t, err)
	secRegex, err := regexp.Compile(` .{2}s `)
	require.Nil(t, err)

	for i, message := range messages {
		message = strings.TrimSpace(message)
		message = podNameSuffixRegex.ReplaceAllString(message, "-XXX ")
		message = eventNameSuffixRegex.ReplaceAllString(message, "-XXX.YYY ")
		message = atBlockRegex.ReplaceAllString(message, "(at some time ago)")
		message = secRegex.ReplaceAllString(message, " ZZs ")
		assert.Equal(t, expectedMessages[i], message)
	}
}
