package store

import (
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

	name := EntityName{Name: "ent1"}

	require.Equal(t, 0, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))
	require.True(t, clusterStore.TryAdd(name, "m", now))
	require.Equal(t, 1, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))
	require.False(t, clusterStore.TryAdd(name, "m", now))
	require.Equal(t, 1, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))
	nearFuture := now.Add(time.Second * time.Duration(50))
	require.False(t, clusterStore.TryAdd(name, "m", nearFuture))
	require.Equal(t, 1, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))
	farFuture := now.Add(time.Minute * time.Duration(2))
	require.True(t, clusterStore.TryAdd(name, "m", farFuture))
	require.Equal(t, 1, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))

	require.True(t, clusterStore.TryAdd(name, "message", farFuture))
	require.Equal(t, 2, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))
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

	name := EntityName{Name: "ent1"}

	require.True(t, clusterStore.TryAdd(name, "a", now))
	require.True(t, clusterStore.TryAdd(name, "b", now))
	require.True(t, clusterStore.TryAdd(name, "c", now))
	require.Equal(t, 3, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))

	storeReloaded, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded := storeReloaded.GetClusterStore("test", now)
	require.Equal(t, 0, len(clusterStoreReloaded.MessagesWithTimestampPerEntity[name.String()]))

	err = store.Flush(now)
	require.Nil(t, err)

	storeReloaded, err = LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded = storeReloaded.GetClusterStore("test", now)
	require.Equal(t, 3, len(clusterStoreReloaded.MessagesWithTimestampPerEntity[name.String()]))
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

	name := EntityName{Name: "ent1"}

	require.True(t, clusterStore.TryAdd(name, "a", now))
	require.True(t, clusterStore.TryAdd(name, "b", now))
	require.True(t, clusterStore.TryAdd(name, "c", now))
	require.Equal(t, 3, len(clusterStore.MessagesWithTimestampPerEntity[name.String()]))

	err = store.Flush(now)
	require.Nil(t, err)

	storeReloaded, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded := storeReloaded.GetClusterStore("test", now.Add(time.Second*time.Duration(50)))
	require.Equal(t, 3, len(clusterStoreReloaded.MessagesWithTimestampPerEntity[name.String()]))
	err = storeReloaded.Flush(now)
	require.Nil(t, err)

	storeReloaded, err = LoadOrCreate(cfg)
	require.Nil(t, err)
	clusterStoreReloaded = storeReloaded.GetClusterStore("test", now.Add(time.Minute*time.Duration(3)))
	require.Equal(t, 0, len(clusterStoreReloaded.MessagesWithTimestampPerEntity[name.String()]))
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

	name := EntityName{Name: "ent1"}

	require.True(t, cluster1Store1.TryAdd(name, "a", now))
	require.True(t, cluster1Store1.TryAdd(name, "b", now))
	require.True(t, cluster1Store1.TryAdd(name, "c", now))
	require.Equal(t, 3, len(cluster1Store1.MessagesWithTimestampPerEntity[name.String()]))

	err = store1.Flush(now)
	require.Nil(t, err)

	store2, err := LoadOrCreate(cfg)
	require.Nil(t, err)
	cluster1Store2 := store2.GetClusterStore("test-1", now)
	require.Equal(t, 3, len(cluster1Store2.MessagesWithTimestampPerEntity[name.String()]))
	cluster2Store2 := store2.GetClusterStore("test-2", now)
	require.Equal(t, 0, len(cluster2Store2.MessagesWithTimestampPerEntity[name.String()]))
	err = store2.Flush(now)
	require.Nil(t, err)
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

	name1 := EntityName{
		Name:      "po1",
		Kind:      "Pod",
		Namespace: "ns",
	}
	require.True(t, clusterStore.TryAdd(name1, "a", now))
	require.True(t, clusterStore.TryAdd(name1, "b", now))
	require.True(t, clusterStore.TryAdd(name1, "c", now))
	require.Equal(t, 3, len(clusterStore.MessagesWithTimestampPerEntity[name1.String()]))

	name2 := EntityName{
		Name: "ns",
		Kind: "Namespace",
	}
	require.True(t, clusterStore.TryAdd(name2, "a", now))
	require.Equal(t, 1, len(clusterStore.MessagesWithTimestampPerEntity[name2.String()]))

	err = store.Flush(now.Add(time.Minute))
	require.Nil(t, err)

	content, err := ioutil.ReadFile(storeFile.Name())
	require.Nil(t, err)
	// language=json
	expectedContent := `{
 "cluster_stores_by_name": {
  "test-json": {
   "cluster": "test-json",
   "messages_with_timestamp_per_entity": {
    "Namespace/ns": {
     "a": "2021-10-17T13:00:00Z"
    },
    "Pod/ns/po1": {
     "a": "2021-10-17T13:00:00Z",
     "b": "2021-10-17T13:00:00Z",
     "c": "2021-10-17T13:00:00Z"
    }
   }
  }
 },
 "last_run_at": "2021-10-17T13:01:00Z"
}`
	require.Equal(t, expectedContent, string(content))
}
