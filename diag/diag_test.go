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
	"sort"
	"testing"
	"time"
)

var apiResponsesDirectoryPath string

func init() {
	_, filePath, _, _ := runtime.Caller(0)
	apiResponsesDirectoryPath = path.Join(filePath, "../../test-resources/api-responses")
}

func setUp(t *testing.T, resourcesDirectoryName string) (*config.Config, kubeclient.KubernetesClient) {
	cfg, err := config.DefaultConfig()
	require.Nil(t, err)

	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)
	cfg.StoreFilePath = storeFile.Name()
	cfg.MessagesDeduplicationDuration = time.Hour

	client, err := kubeclient.CreateMockClient(
		path.Join(apiResponsesDirectoryPath, resourcesDirectoryName, "nodes.json"),
		path.Join(apiResponsesDirectoryPath, resourcesDirectoryName, "ns.json"),
		path.Join(apiResponsesDirectoryPath, resourcesDirectoryName, "pods.json"),
		path.Join(apiResponsesDirectoryPath, resourcesDirectoryName, "rs.json"),
		path.Join(apiResponsesDirectoryPath, resourcesDirectoryName, "events.json"),
	)
	require.Nil(t, err)
	require.NotNil(t, client)
	return cfg, client
}

func Test_Diagnose(t *testing.T) {
	cfg, client := setUp(t, "integration-test-outputs")

	now := asTime("2021-10-17T14:20:00Z")

	stor, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStore := stor.GetClusterStore("diag-test", now)
	assert.Equal(t, 0, len(clusterStore.Alerts))
	err = DiagnoseCluster(client, cfg, clusterStore, now)
	require.Nil(t, err)
	err = stor.Flush(now)
	require.Nil(t, err)

	alerts := clusterStore.Alerts
	sort.Sort(alerts)
	assert.Equal(t, 5, len(alerts))

	i := 0
	assert.Equal(t, clusterStore.Cluster, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-2-broken-image-7cbf974df9-4jv8f", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 1, len(alerts[i].Messages))
	assert.Equal(t, "Container test-2-broken-image still waiting due to ImagePullBackOff: Back-off pulling image \"nginx:l4t3st\"", alerts[i].Messages[0])
	assert.Equal(t, 2, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: Failed x4 since 17 Oct 21 14:15 UTC, 4 minutes ago (last seen 2 minutes ago):
	Failed to pull image "nginx:l4t3st": rpc error: code = Unknown desc = Error response from daemon: manifest for nginx:l4t3st not found: manifest unknown: manifest unknown`, alerts[i].Events[0])
	assert.Equal(t, `Event by kubelet: Failed x6 since 17 Oct 21 14:15 UTC, 4 minutes ago (last seen 2 minutes ago):
	Error: ImagePullBackOff`, alerts[i].Events[1])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-2-broken-image-7cbf974df9-4jv8f/test-2-broken-image/logs", alerts[i].LogsByContainerName["test-2-broken-image"])

	i++
	assert.Equal(t, clusterStore.Cluster, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-3-excessive-resources-699d58f55f-q9z65", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 1, len(alerts[i].Messages))
	assert.Equal(t, "Unschedulable: 0/1 nodes are available: 1 Insufficient memory. (last transition: 4 minutes ago)", alerts[i].Messages[0])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by default-scheduler: FailedScheduling since 17 Oct 21 14:16 UTC, 3 minutes ago (last seen 2 minutes ago):
	0/1 nodes are available: 1 Insufficient memory.`, alerts[i].Events[0])
	assert.Equal(t, 0, len(alerts[i].LogsByContainerName))

	i++
	assert.Equal(t, clusterStore.Cluster, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-4-crashlooping-dbdd84589-8m7kj", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 1, len(alerts[i].Messages))
	assert.Equal(t, "Container test-4-crashlooping is in CrashLoopBackOff: restarted 4 times, last exit due to Error (exit code 1)", alerts[i].Messages[0])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: BackOff x8 since 17 Oct 21 14:15 UTC, 4 minutes ago (last seen 3 minutes ago):
	Back-off restarting failed container`, alerts[i].Events[0])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-4-crashlooping-dbdd84589-8m7kj/test-4-crashlooping/logs", alerts[i].LogsByContainerName["test-4-crashlooping"])

	i++
	assert.Equal(t, clusterStore.Cluster, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-5-completed-757685986-qxbqp", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 1, len(alerts[i].Messages))
	assert.Equal(t, "Container test-5-completed is in CrashLoopBackOff: restarted 4 times, last exit due to Completed (exit code 0)", alerts[i].Messages[0])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: BackOff x8 since 17 Oct 21 14:15 UTC, 4 minutes ago (last seen 2 minutes ago):
	Back-off restarting failed container`, alerts[i].Events[0])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-5-completed-757685986-qxbqp/test-5-completed/logs", alerts[i].LogsByContainerName["test-5-completed"])

	i++
	assert.Equal(t, clusterStore.Cluster, alerts[i].ClusterName)
	assert.Equal(t, "default", alerts[i].Namespace)
	assert.Equal(t, "test-6-crashlooping-init-644545f5b7-l468n", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 1, len(alerts[i].Messages))
	assert.Equal(t, "Container test-6-crashlooping-init-container (init) is in CrashLoopBackOff: restarted 4 times, last exit due to Error (exit code 1)", alerts[i].Messages[0])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: BackOff x7 since 17 Oct 21 14:15 UTC, 4 minutes ago (last seen 3 minutes ago):
	Back-off restarting failed container`, alerts[i].Events[0])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "default/test-6-crashlooping-init-644545f5b7-l468n/test-6-crashlooping-init-container/logs", alerts[i].LogsByContainerName["test-6-crashlooping-init-container"])
}

func Test_Diagnose_RepeatingCallAfterShortTime(t *testing.T) {
	cfg, client := setUp(t, "integration-test-outputs")

	now := asTime("2021-10-17T14:20:00Z")
	clusterName := "diag-test-short"

	store1, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStore1 := store1.GetClusterStore(clusterName, now)
	assert.Equal(t, 0, len(clusterStore1.Alerts))
	err = DiagnoseCluster(client, cfg, clusterStore1, now)
	require.Nil(t, err)
	assert.Equal(t, 5, len(clusterStore1.Alerts))
	err = store1.Flush(now)
	require.Nil(t, err)

	nearFuture := now.Add(time.Minute)

	store2, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	clusterStore2 := store2.GetClusterStore(clusterName, nearFuture)

	err = DiagnoseCluster(client, cfg, clusterStore2, nearFuture)
	require.Nil(t, err)
	assert.Equal(t, 0, len(clusterStore2.Alerts))
	err = store2.Flush(nearFuture)
	require.Nil(t, err)
}

func Test_Diagnose_RepeatingCallAfterLongTime(t *testing.T) {
	cfg, client := setUp(t, "integration-test-outputs")

	now := asTime("2021-10-17T14:20:00Z")
	clusterName := "diag-test-long"

	store1, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStore1 := store1.GetClusterStore(clusterName, now)
	assert.Equal(t, 0, len(clusterStore1.Alerts))
	err = DiagnoseCluster(client, cfg, clusterStore1, now)
	require.Nil(t, err)
	assert.Equal(t, 5, len(clusterStore1.Alerts))
	err = store1.Flush(now)
	require.Nil(t, err)

	farFuture := now.Add(time.Hour + time.Minute)

	store2, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)

	clusterStore2 := store2.GetClusterStore(clusterName, farFuture)

	err = DiagnoseCluster(client, cfg, clusterStore2, farFuture)
	require.Nil(t, err)
	assert.Equal(t, 5, len(clusterStore2.Alerts))
	err = store2.Flush(farFuture)
	require.Nil(t, err)
}

func Test_Diagnose_EventsDeduplicationOnLivenessCheckFail(t *testing.T) {
	cfg, client := setUp(t, "liveness-fails")

	now := asTime("2021-10-31T14:30:00Z")
	clusterName := "diag-test-events-liveness-fails"

	stor, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStore := stor.GetClusterStore(clusterName, now)
	assert.Equal(t, 0, len(clusterStore.Alerts))
	err = DiagnoseCluster(client, cfg, clusterStore, now)
	require.Nil(t, err)

	alerts := clusterStore.Alerts
	sort.Sort(alerts)
	assert.Equal(t, 3, len(alerts))

	err = stor.Flush(now)
	require.Nil(t, err)

	i := 0
	assert.Equal(t, clusterStore.Cluster, alerts[i].ClusterName)
	assert.Equal(t, "dd9bf8cf4edf444589e69aaa05", alerts[i].Namespace)
	assert.Equal(t, "app2-b97ccc4f-9p8d8", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 1, len(alerts[i].Messages))
	assert.Equal(t, "Container app2 restarted 10 times, last exit due to Error (exit code 137)", alerts[i].Messages[0])
	assert.Equal(t, 1, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: Unhealthy x155 since 31 Oct 21 08:03 UTC, 6 hours ago (last seen now):
	Liveness probe failed:   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
	Dload  Upload   Total   Spent    Left  Speed
	0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0
	0     0    0     0    0     0      0      0 --:--:--  0:00:01 --:--:--     0
	0     0    0     0    0     0      0      0 --:--:--  0:00:02 --:--:--     0
	0     0    0     0    0     0      0      0 --:--:--  0:00:03 --:--:--     0
	curl: (28) Operation timed out after 3001 milliseconds with 0 bytes received`, alerts[i].Events[0])
	assert.Equal(t, 1, len(alerts[i].LogsByContainerName))
	assert.Equal(t, "dd9bf8cf4edf444589e69aaa05/app2-b97ccc4f-9p8d8/app2/logs", alerts[i].LogsByContainerName["app2"])

	stor, err = store.LoadOrCreate(cfg)
	require.Nil(t, err)
	now = now.Add(time.Minute  * time.Duration(17))
	clusterStore = stor.GetClusterStore(clusterName, now)
	assert.Equal(t, 0, len(clusterStore.Alerts))
	err = DiagnoseCluster(client, cfg, clusterStore, now)
	require.Nil(t, err)

	alerts = clusterStore.Alerts
	assert.Equal(t, 0, len(alerts))
}

func Test_Diagnose_EventsDeduplicationOnRpcError(t *testing.T) {
	cfg, client := setUp(t, "rpc-error")

	now := asTime("2021-10-31T14:30:00Z")
	clusterName := "diag-test-events-rpc-error"

	stor, err := store.LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStore := stor.GetClusterStore(clusterName, now)
	assert.Equal(t, 0, len(clusterStore.Alerts))
	err = DiagnoseCluster(client, cfg, clusterStore, now)
	require.Nil(t, err)

	alerts := clusterStore.Alerts
	sort.Sort(alerts)
	assert.Equal(t, 2, len(alerts))

	err = stor.Flush(now)
	require.Nil(t, err)

	i := 0
	assert.Equal(t, clusterStore.Cluster, alerts[i].ClusterName)
	assert.Equal(t, "ci", alerts[i].Namespace)
	assert.Equal(t, "api-74767b9df-xxsrs", alerts[i].Name)
	assert.Equal(t, "Pod", alerts[i].Kind)
	assert.Equal(t, 1, len(alerts[i].Messages))
	assert.Equal(t, "4 containers are still initializing [ init-container (init), run-migrations (init), wait-for-database (init), wait-for-queue (init) ] (since 1 hour ago)", alerts[i].Messages[0])
	assert.Equal(t, 2, len(alerts[i].Events))
	assert.Equal(t, `Event by kubelet: FailedCreatePodSandBox x2 since 31 Oct 21 14:21 UTC, 8 minutes ago (last seen 7 minutes ago):
	Failed to create pod sandbox: rpc error: code = Unknown desc = failed to reserve sandbox name "api-74767b9df-xxsrs_ci_6e8c94db-df75-480c-824b-92eb95e99296_0": name "api-74767b9df-xxsrs_ci_6e8c94db-df75-480c-824b-92eb95e99296_0" is reserved for "067d2ab2c9c553f48b0294d2b07926ea278a6b1ad74c716dd59b43cf3d2ca6e9"`, alerts[i].Events[0])
	assert.Equal(t, `Event by kubelet: FailedCreatePodSandBox x18 since 31 Oct 21 13:32 UTC, 57 minutes ago (last seen now):
	Failed to create pod sandbox: rpc error: code = DeadlineExceeded desc = context deadline exceeded`, alerts[i].Events[1])
	assert.Equal(t, 0, len(alerts[i].LogsByContainerName))

	stor, err = store.LoadOrCreate(cfg)
	require.Nil(t, err)
	now = now.Add(time.Minute  * time.Duration(17))
	clusterStore = stor.GetClusterStore(clusterName, now)
	assert.Equal(t, 0, len(clusterStore.Alerts))
	err = DiagnoseCluster(client, cfg, clusterStore, now)
	require.Nil(t, err)

	alerts = clusterStore.Alerts
	assert.Equal(t, 0, len(alerts))
}
