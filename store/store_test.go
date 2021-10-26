package store

import (
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
	"time"
)

func TestStoreCreateAndFlush(t *testing.T) {
	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	cfg := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
	}
	store, err := LoadOrCreate(cfg)
	require.Nil(t, err)

	content, err := ioutil.ReadFile(storeFile.Name())
	require.Nil(t, err)
	require.True(t, len(content) == 0)

	err = store.Flush(time.Now())
	require.Nil(t, err)

	content, err = ioutil.ReadFile(storeFile.Name())
	require.Nil(t, err)
	require.True(t, len(content) > 0)
}

func TestStoreAddFlow(t *testing.T) {
	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)
	now := time.Now().UTC()

	cfg := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
	}
	store, err := LoadOrCreate(cfg)
	require.Nil(t, err)

	clusterStore := store.GetClusterStore("test", now)

	require.Equal(t, 0, len(clusterStore.Alerts))
	require.True(t, clusterStore.ShouldAdd("hash1", now))
	clusterStore.Add(&alert.EntityAlert{Messages: []string{"message1"}}, []string{"hash1"}, now)
	require.Equal(t, 1, len(clusterStore.Alerts))
	require.False(t, clusterStore.ShouldAdd("hash1", now))
	require.Equal(t, 1, len(clusterStore.Alerts))
	nearFuture := now.Add(time.Second * time.Duration(50))
	require.False(t, clusterStore.ShouldAdd("hash1", nearFuture))
	require.Equal(t, 1, len(clusterStore.Alerts))
	farFuture := now.Add(time.Minute * time.Duration(2))
	require.True(t, clusterStore.ShouldAdd("hash1", farFuture))
	clusterStore.Add(&alert.EntityAlert{Messages: []string{"message1"}}, []string{"hash1"}, farFuture)
	require.Equal(t, 2, len(clusterStore.Alerts))

	require.True(t, clusterStore.ShouldAdd("hash2", now))
	clusterStore.Add(&alert.EntityAlert{Messages: []string{"message2"}}, []string{"hash2"}, now)
	require.Equal(t, 3, len(clusterStore.Alerts))
	require.True(t, clusterStore.ShouldAdd("hash3", now))
	clusterStore.Add(&alert.EntityAlert{Messages: []string{"message2"}}, []string{"hash3"}, now)
	require.Equal(t, 4, len(clusterStore.Alerts))
	require.False(t, clusterStore.ShouldAdd("hash3", now))
	require.Equal(t, 4, len(clusterStore.Alerts))
}

func TestLoadAfterFlush(t *testing.T) {
	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)
	now := time.Now().UTC()

	cfg := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
	}
	store, err := LoadOrCreate(cfg)
	require.Nil(t, err)

	clusterStore := store.GetClusterStore("test", now)

	clusterStore.Add(&alert.EntityAlert{Messages: []string{"message"}}, []string{"hash1", "hash2", "hash3"}, now)
	require.Equal(t, 1, len(clusterStore.Alerts))
	require.Equal(t, 3, len(clusterStore.HashWithTimestamp))

	storeReloaded, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded := storeReloaded.GetClusterStore("test", now)
	require.Equal(t, 0, len(clusterStoreReloaded.Alerts))
	require.Equal(t, 0, len(clusterStoreReloaded.HashWithTimestamp))

	err = store.Flush(now)
	require.Nil(t, err)

	storeReloaded, err = LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded = storeReloaded.GetClusterStore("test", now)
	require.Equal(t, 0, len(clusterStoreReloaded.Alerts))
	require.Equal(t, 3, len(clusterStoreReloaded.HashWithTimestamp))
}

func TestLoadAfterLongTime(t *testing.T) {
	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)
	now := time.Now().UTC()

	cfg := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
	}
	store, err := LoadOrCreate(cfg)
	require.Nil(t, err)

	clusterStore := store.GetClusterStore("test", now)

	clusterStore.Add(&alert.EntityAlert{Messages: []string{"message"}}, []string{"hash1", "hash2", "hash3"}, now)
	require.Equal(t, 1, len(clusterStore.Alerts))
	require.Equal(t, 3, len(clusterStore.HashWithTimestamp))
	err = store.Flush(now)
	require.Nil(t, err)

	storeReloaded, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded := storeReloaded.GetClusterStore("test", now.Add(time.Second*time.Duration(50)))
	require.Equal(t, 0, len(clusterStoreReloaded.Alerts))
	require.Equal(t, 3, len(clusterStoreReloaded.HashWithTimestamp))
	err = storeReloaded.Flush(now)
	require.Nil(t, err)

	storeReloaded, err = LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded = storeReloaded.GetClusterStore("test", now.Add(time.Minute*time.Duration(3)))
	require.Equal(t, 0, len(clusterStoreReloaded.Alerts))
	require.Equal(t, 0, len(clusterStoreReloaded.HashWithTimestamp))
	err = storeReloaded.Flush(now)
	require.Nil(t, err)
}

func TestStoreForMultipleClusters(t *testing.T) {
	now := time.Now().UTC()
	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	cfg := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
	}
	store1, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	cluster1Store1 := store1.GetClusterStore("test-1", now)

	cluster1Store1.Add(&alert.EntityAlert{Messages: []string{"message"}}, []string{"hash1", "hash2", "hash3"}, now)
	require.Equal(t, 1, len(cluster1Store1.Alerts))
	require.Equal(t, 3, len(cluster1Store1.HashWithTimestamp))
	err = store1.Flush(now)
	require.Nil(t, err)

	store2, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	cluster2Store2 := store2.GetClusterStore("test-2", now)
	require.Equal(t, 0, len(cluster2Store2.Alerts))
	require.Equal(t, 0, len(cluster2Store2.HashWithTimestamp))
	err = store2.Flush(now)
	require.Nil(t, err)

	require.Equal(t, 2, len(store2.ClusterStoresByName))

	store3, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	cluster3Store3 := store3.GetClusterStore("test-3", now)
	require.Nil(t, err)
	require.Equal(t, 0, len(cluster3Store3.Alerts))
	require.Equal(t, 0, len(cluster3Store3.HashWithTimestamp))
	err = store3.Flush(now)
	require.Nil(t, err)

	require.Equal(t, 1, len(store1.ClusterStoresByName))
	require.Equal(t, 2, len(store2.ClusterStoresByName))
	require.Equal(t, 3, len(store3.ClusterStoresByName))
}

func TestJsonContent(t *testing.T) {
	time.Local = time.UTC
	now, err := time.Parse(time.RFC822, "17 Oct 21 13:00 IDT")
	require.Nil(t, err)

	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	cfg := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
	}
	store, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStore := store.GetClusterStore("test-json", now)

	clusterStore.Add(&alert.EntityAlert{Messages: []string{"message"}}, []string{"hash1", "hash2", "hash3"}, now)
	require.Equal(t, 1, len(clusterStore.Alerts))
	err = store.Flush(now.Add(time.Minute))
	require.Nil(t, err)

	content, err := ioutil.ReadFile(storeFile.Name())
	require.Nil(t, err)
	// language=json
	expectedContent := `{
 "cluster_stores_by_name": {
  "test-json": {
   "cluster": "test-json",
   "hash_with_timestamp": {
    "hash1": "2021-10-17T13:00:00Z",
    "hash2": "2021-10-17T13:00:00Z",
    "hash3": "2021-10-17T13:00:00Z"
   }
  }
 },
 "last_run_at": "2021-10-17T13:01:00Z"
}`
	require.Equal(t, expectedContent, string(content))
}
