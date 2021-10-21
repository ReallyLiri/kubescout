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

	configuration := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
		ClusterName:                   "test",
	}
	store, err := LoadOrCreate(configuration)
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

	configuration := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
		ClusterName:                   "test",
	}
	store, err := LoadOrCreate(configuration)
	require.Nil(t, err)

	now := time.Now().UTC()
	require.Equal(t, 0, len(store.RelevantMessages()))
	require.True(t, store.ShouldAdd("hash1", now))
	store.Add("message1", []string{"hash1"}, now)
	require.Equal(t, 1, len(store.RelevantMessages()))
	require.False(t, store.ShouldAdd("hash1", now))
	require.Equal(t, 1, len(store.RelevantMessages()))
	nearFuture := now.Add(time.Second * time.Duration(50))
	require.False(t, store.ShouldAdd("hash1", nearFuture))
	require.Equal(t, 1, len(store.RelevantMessages()))
	farFuture := now.Add(time.Minute * time.Duration(2))
	require.True(t, store.ShouldAdd("hash1", farFuture))
	store.Add("message1", []string{"hash1"}, farFuture)
	require.Equal(t, 2, len(store.RelevantMessages()))

	require.True(t, store.ShouldAdd("hash2", now))
	store.Add("message2", []string{"hash2"}, now)
	require.Equal(t, 3, len(store.RelevantMessages()))
	require.True(t, store.ShouldAdd("hash3", now))
	store.Add("message2", []string{"hash3"}, now)
	require.Equal(t, 4, len(store.RelevantMessages()))
	require.False(t, store.ShouldAdd("hash3", now))
	require.Equal(t, 4, len(store.RelevantMessages()))
}

func TestLoadAfterFlush(t *testing.T) {
	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	configuration := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
		ClusterName:                   "test",
	}
	store, err := LoadOrCreate(configuration)
	require.Nil(t, err)

	now := time.Now().UTC()
	store.Add("message", []string{"hash1", "hash2", "hash3"}, now)
	require.Equal(t, 1, len(store.RelevantMessages()))
	require.Equal(t, 3, len(store.ClusterStoresByName["test"].HashWithTimestamp))

	require.True(t, store.IsRelevant(now.Add(time.Duration(-1) * time.Minute)))
	require.True(t, store.IsRelevant(now.Add(time.Minute)))

	storeReloaded, err := LoadOrCreate(configuration)
	require.Nil(t, err)
	require.Equal(t, 0, len(storeReloaded.RelevantMessages()))
	require.Equal(t, 0, len(storeReloaded.ClusterStoresByName["test"].HashWithTimestamp))

	err = store.Flush(now)
	require.Nil(t, err)

	storeReloaded, err = LoadOrCreate(configuration)
	require.Nil(t, err)
	require.Equal(t, 0, len(storeReloaded.RelevantMessages()))
	require.Equal(t, 3, len(store.ClusterStoresByName["test"].HashWithTimestamp))

	require.False(t, storeReloaded.IsRelevant(now.Add(time.Duration(-1) * time.Minute)))
	require.True(t, storeReloaded.IsRelevant(now.Add(time.Minute)))
}

func TestStoreForMultipleClusters(t *testing.T) {
	now := time.Now().UTC()
	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	configuration := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
		ClusterName:                   "test-1",
	}
	store1, err := LoadOrCreate(configuration)
	require.Nil(t, err)

	store1.Add("message", []string{"hash1", "hash2", "hash3"}, now)
	require.Equal(t, 1, len(store1.RelevantMessages()))
	require.Equal(t, 3, len(store1.ClusterStoresByName["test-1"].HashWithTimestamp))
	err = store1.Flush(now)
	require.Nil(t, err)

	configuration.ClusterName = "test-2"
	store2, err := LoadOrCreate(configuration)
	require.Nil(t, err)
	require.Equal(t, 0, len(store2.RelevantMessages()))
	require.Equal(t, 0, len(store2.ClusterStoresByName["test-2"].HashWithTimestamp))
	err = store2.Flush(now)
	require.Nil(t, err)

	require.Equal(t, 2, len(store2.ClusterStoresByName))

	configuration.ClusterName = "test-3"
	store3, err := LoadOrCreate(configuration)
	require.Nil(t, err)
	require.Equal(t, 0, len(store3.RelevantMessages()))
	require.Equal(t, 0, len(store3.ClusterStoresByName["test-3"].HashWithTimestamp))
	err = store2.Flush(now)
	require.Nil(t, err)

	require.Equal(t, 3, len(store3.ClusterStoresByName))
}

func TestJsonContent(t *testing.T) {
	time.Local = time.UTC
	now, err := time.Parse(time.RFC822, "17 Oct 21 13:00 IDT")
	require.Nil(t, err)

	storeFile, err := ioutil.TempFile(t.TempDir(), "*.store.json")
	require.Nil(t, err)

	configuration := &config.Config{
		StoreFilePath:                 storeFile.Name(),
		MessagesDeduplicationDuration: time.Minute,
		ClusterName:                   "test-json",
	}
	store1, err := LoadOrCreate(configuration)
	require.Nil(t, err)

	store1.Add("message", []string{"hash1", "hash2", "hash3"}, now)
	require.Equal(t, 1, len(store1.RelevantMessages()))
	err = store1.Flush(now.Add(time.Minute))
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
   },
   "last_run_at": "2021-10-17T13:01:00Z"
  }
 }
}`
	require.Equal(t, expectedContent, string(content))
}
