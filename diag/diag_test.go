package diag

import (
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/kubeclient"
	"github.com/reallyliri/kubescout/store"
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
	assert.Equal(t, 0, len(sto.EntityAlerts()))
	err = DiagnoseCluster(client, cfg, sto, now)
	require.Nil(t, err)
	err = sto.Flush(now)
	require.Nil(t, err)

	alerts := sto.EntityAlerts()
	assert.Equal(t, 6, len(alerts))

	i := 0
	assert.Equal(t, cfg.ClusterName, alerts[i].ClusterName)
	assert.Equal(t, "", alerts[i].Namespace)
	assert.Equal(t, "minikube", alerts[i].Name)
	assert.Equal(t, "Node", alerts[i].Kind)
	assert.Equal(t, 0, len(alerts[i].Messages))
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by sysctl-monitor: NodeSysctlChange x53 since 17 Oct 21 14:15 UTC (last seen 4 minutes ago)`, alerts[i].Events[0])
	assert.Equal(t, 0, len(alerts[i].LogsByContainerName))

	i = 1
	assert.Equal(t, cfg.ClusterName, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-2-broken-image-7cbf974df9-4jv8f", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 2, len(alerts[i].Messages))
	assert.Equal(t, "Pod is in Pending phase", alerts[i].Messages[0])
	assert.Equal(t, "test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image \"nginx:l4t3st\"", alerts[i].Messages[1])
	assert.Equal(t, 3, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: Failed x4 since 17 Oct 21 14:15 UTC (last seen 2 minutes ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`, alerts[i].Events[0])
	assert.Equal(t, `Event by kubelet: Failed x4 since 17 Oct 21 14:15 UTC (last seen 2 minutes ago):
	Error: ErrImagePull`, alerts[i].Events[1])
	assert.Equal(t, `Event by kubelet: Failed x6 since 17 Oct 21 14:15 UTC (last seen 2 minutes ago):
	Error: ImagePullBackOff`, alerts[i].Events[2])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-2-broken-image-7cbf974df9-4jv8f/test-2-broken-image/logs", alerts[i].LogsByContainerName["test-2-broken-image"])

	i = 2
	assert.Equal(t, cfg.ClusterName, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-3-excessive-resources-699d58f55f-q9z65", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 2, len(alerts[i].Messages))
	assert.Equal(t, "Pod is in Pending phase", alerts[i].Messages[0])
	assert.Equal(t, "Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 4 minutes ago)", alerts[i].Messages[1])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by default-scheduler: FailedScheduling since 17 Oct 21 14:15 UTC (last seen 4 minutes ago):
	0/1 nodes are available: 1 Insufficient memory.`, alerts[i].Events[0])
	assert.Equal(t, 0, len(alerts[i].LogsByContainerName))

	i = 3
	assert.Equal(t, cfg.ClusterName, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-4-crashlooping-dbdd84589-8m7kj", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 2, len(alerts[i].Messages))
	assert.Equal(t, "test-4-crashlooping still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container", alerts[i].Messages[0])
	assert.Equal(t, "test-4-crashlooping had restarted 4 times, last exit due to Error (exit code 1)", alerts[i].Messages[1])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: BackOff x8 since 17 Oct 21 14:15 UTC (last seen 3 minutes ago):
	Back-off restarting failed container`, alerts[i].Events[0])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-4-crashlooping-dbdd84589-8m7kj/test-4-crashlooping/logs", alerts[i].LogsByContainerName["test-4-crashlooping"])

	i = 4
	assert.Equal(t, cfg.ClusterName, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-5-completed-757685986-qxbqp", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 2, len(alerts[i].Messages))
	assert.Equal(t, "test-5-completed still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container", alerts[i].Messages[0])
	assert.Equal(t, "test-5-completed had restarted 4 times, last exit due to Completed (exit code 0)", alerts[i].Messages[1])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: BackOff x8 since 17 Oct 21 14:15 UTC (last seen 2 minutes ago):
	Back-off restarting failed container`, alerts[i].Events[0])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-5-completed-757685986-qxbqp/test-5-completed/logs", alerts[i].LogsByContainerName["test-5-completed"])

	i = 5
	assert.Equal(t, cfg.ClusterName, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-6-crashlooping-init-644545f5b7-l468n", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 3, len(alerts[i].Messages))
	assert.Equal(t, "Pod is in Pending phase", alerts[i].Messages[0])
	assert.Equal(t, "test-6-crashlooping-init-container (init) still waiting due to CrashLoopBackOff: back-off 1m20s restarting failed container", alerts[i].Messages[1])
	assert.Equal(t, "test-6-crashlooping-init-container (init) had restarted 4 times, last exit due to Error (exit code 1)", alerts[i].Messages[2])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: BackOff x7 since 17 Oct 21 14:15 UTC (last seen 3 minutes ago):
	Back-off restarting failed container`, alerts[i].Events[0])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-6-crashlooping-init-644545f5b7-l468n/test-6-crashlooping-init-container/logs", alerts[i].LogsByContainerName["test-6-crashlooping-init-container"])
}

func TestDiagnoseRepeatingCallAfterShortTime(t *testing.T) {
	cfg, client := setUp(t)

	now := asTime("2021-10-17T14:20:00Z")

	store1, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	assert.Equal(t, 0, len(store1.EntityAlerts()))
	err = DiagnoseCluster(client, cfg, store1, now)
	require.Nil(t, err)
	assert.Equal(t, 6, len(store1.EntityAlerts()))
	err = store1.Flush(now)
	require.Nil(t, err)

	store2, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	nearFuture := now.Add(time.Minute)
	err = DiagnoseCluster(client, cfg, store2, nearFuture)
	require.Nil(t, err)
	assert.Equal(t, 0, len(store2.EntityAlerts()))
	err = store2.Flush(now)
	require.Nil(t, err)
}

func TestDiagnoseRepeatingCallAfterLongTime(t *testing.T) {
	cfg, client := setUp(t)

	now := asTime("2021-10-17T14:20:00Z")

	store1, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	assert.Equal(t, 0, len(store1.EntityAlerts()))
	err = DiagnoseCluster(client, cfg, store1, now)
	require.Nil(t, err)
	assert.Equal(t, 6, len(store1.EntityAlerts()))
	err = store1.Flush(now)
	require.Nil(t, err)

	store2, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	farFuture := now.Add(time.Hour + time.Minute)
	err = DiagnoseCluster(client, cfg, store2, farFuture)
	require.Nil(t, err)
	assert.Equal(t, 6, len(store2.EntityAlerts()))
	err = store2.Flush(now)
	require.Nil(t, err)
}
