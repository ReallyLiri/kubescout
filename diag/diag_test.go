package diag

import (
	"KubeScout/config"
	"KubeScout/kubeclient"
	"KubeScout/store"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
	"time"
)

var apiResponsesDirectoryPath string

func init() {
	_, filePath, _, _ := runtime.Caller(0)
	apiResponsesDirectoryPath = path.Join(filePath, "../../test-resources/api-responses")
}

func setUp(t *testing.T) (*config.Config, kubeclient.KubernetesClient) {
	cfg, err := config.DefaultConfig()
	require.Nil(t, err)

	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)
	cfg.ClusterName = "diag-integration"
	cfg.StoreFilePath = storeFile.Name()
	cfg.MessagesDeduplicationDuration = time.Hour

	client, err := kubeclient.CreateMockClient(
		path.Join(apiResponsesDirectoryPath, "integration-test-outputs", "nodes.json"),
		path.Join(apiResponsesDirectoryPath, "integration-test-outputs", "ns.json"),
		path.Join(apiResponsesDirectoryPath, "integration-test-outputs", "pods.json"),
		path.Join(apiResponsesDirectoryPath, "integration-test-outputs", "rs.json"),
		path.Join(apiResponsesDirectoryPath, "integration-test-outputs", "events.json"),
	)
	require.Nil(t, err)
	require.NotNil(t, client)
	return cfg, client
}

func TestDiagnose(t *testing.T) {
	cfg, client := setUp(t)

	now := asTime("2021-10-17T14:20:00Z")

	sto, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	assert.Equal(t, 0, len(sto.RelevantMessages()))
	err = DiagnoseCluster(client, cfg, sto, now)
	require.Nil(t, err)

	messages := sto.RelevantMessages()
	assert.Equal(t, 12, len(messages))
	for i, message := range messages {
		fmt.Printf("checking message #%v ...\n", i)
		fmt.Printf(message + "\n")
		assert.Equal(t, expectedMessages[i], message)
	}
}

func TestDiagnoseRepeatingCallAfterShortTime(t *testing.T) {
	cfg, client := setUp(t)

	now := asTime("2021-10-17T14:20:00Z")

	store1, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	assert.Equal(t, 0, len(store1.RelevantMessages()))
	err = DiagnoseCluster(client, cfg, store1, now)
	require.Nil(t, err)
	assert.Equal(t, 12, len(store1.RelevantMessages()))

	store2, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	nearFuture := now.Add(time.Minute)
	err = DiagnoseCluster(client, cfg, store2, nearFuture)
	require.Nil(t, err)
	assert.Equal(t, 0, len(store2.RelevantMessages()))
}

func TestDiagnoseRepeatingCallAfterLongTime(t *testing.T) {
	cfg, client := setUp(t)

	now := asTime("2021-10-17T14:20:00Z")

	store1, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	assert.Equal(t, 0, len(store1.RelevantMessages()))
	err = DiagnoseCluster(client, cfg, store1, now)
	require.Nil(t, err)
	assert.Equal(t, 12, len(store1.RelevantMessages()))

	store2, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	farFuture := now.Add(time.Hour + time.Minute)
	err = DiagnoseCluster(client, cfg, store2, farFuture)
	require.Nil(t, err)
	assert.Equal(t, 12, len(store2.RelevantMessages()))
}

var expectedMessages = []string{
	`Pod default/test-2-broken-image-7cbf974df9-4jv8f is un-healthy
	Pod is in Pending phase
	test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image "nginx:l4t3st"
logs of container test-2-broken-image:
<<<<<<<<<<
default/test-2-broken-image-7cbf974df9-4jv8f/test-2-broken-image/logs
>>>>>>>>>>`,

	`Event on Pod test-2-broken-image-7cbf974df9-4jv8f due to Failed (at 17 Oct 21 14:17 UTC, 2 minutes ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`,

	`Event on Pod test-2-broken-image-7cbf974df9-4jv8f due to Failed (at 17 Oct 21 14:17 UTC, 2 minutes ago):
	Error: ErrImagePull`,

	`Event on Pod test-2-broken-image-7cbf974df9-4jv8f due to Failed (at 17 Oct 21 14:17 UTC, 2 minutes ago):
	Error: ImagePullBackOff`,

	`Pod default/test-3-excessive-resources-699d58f55f-q9z65 is un-healthy
	Pod is in Pending phase
	Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 4 minutes ago)`,

	`Event on Pod test-3-excessive-resources-699d58f55f-q9z65 due to FailedScheduling (at unavailable time, unknown time ago):
	0/1 nodes are available: 1 Insufficient memory.`,

	`Pod default/test-4-crashlooping-dbdd84589-8m7kj is un-healthy
	test-4-crashlooping still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container
	test-4-crashlooping had restarted 4 times, last exit due to Error (exit code 1)
logs of container test-4-crashlooping:
<<<<<<<<<<
default/test-4-crashlooping-dbdd84589-8m7kj/test-4-crashlooping/logs
>>>>>>>>>>`,

	`Event on Pod test-4-crashlooping-dbdd84589-8m7kj due to BackOff (at 17 Oct 21 14:16 UTC, 3 minutes ago):
	Back-off restarting failed container`,

	`Pod default/test-5-completed-757685986-qxbqp is un-healthy
	test-5-completed still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container
	test-5-completed had restarted 4 times, last exit due to Completed (exit code 0)
logs of container test-5-completed:
<<<<<<<<<<
default/test-5-completed-757685986-qxbqp/test-5-completed/logs
>>>>>>>>>>`,

	`Event on Pod test-5-completed-757685986-qxbqp due to BackOff (at 17 Oct 21 14:17 UTC, 2 minutes ago):
	Back-off restarting failed container`,

	`Pod default/test-6-crashlooping-init-644545f5b7-l468n is un-healthy
	Pod is in Pending phase
	test-6-crashlooping-init-container (init) still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container
	test-6-crashlooping-init-container (init) had restarted 4 times, last exit due to Error (exit code 1)
logs of container test-6-crashlooping-init-container:
<<<<<<<<<<
default/test-6-crashlooping-init-644545f5b7-l468n/test-6-crashlooping-init-container/logs
>>>>>>>>>>`,

	`Event on Pod test-6-crashlooping-init-644545f5b7-l468n due to BackOff (at 17 Oct 21 14:16 UTC, 3 minutes ago):
	Back-off restarting failed container`,
}
