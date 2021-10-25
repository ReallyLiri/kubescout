package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"io/fs"
	"io/ioutil"
	"time"
)

type Store struct {
	ClusterStoresByName map[string]*ClusterStore `json:"cluster_stores_by_name"`
	dedupDuration       time.Duration
	filePath            string
	LastRunAt           time.Time `json:"last_run_at"`
}

type ClusterStore struct {
	parent            *Store
	Cluster           string               `json:"cluster"`
	HashWithTimestamp map[string]time.Time `json:"hash_with_timestamp"`
	Alerts            alert.EntityAlerts   `json:"-"`
}

func LoadOrCreate(config *config.Config) (*Store, error) {
	store := &Store{
		ClusterStoresByName: make(map[string]*ClusterStore),
		dedupDuration:       config.MessagesDeduplicationDuration,
		filePath:            config.StoreFilePath,
	}
	if store.filePath == "" {
		return store, nil
	}

	content, err := ioutil.ReadFile(store.filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return store, nil
		}
		return nil, fmt.Errorf("failed to read store file from '%v': %v", store.filePath, err)
	}
	if len(content) == 0 {
		return store, nil
	}

	err = json.Unmarshal(content, store)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize json from '%v': %v", store.filePath, err)
	}
	return store, nil
}

func (store *Store) GetClusterStore(name string, now time.Time) *ClusterStore {
	clusterStore, exists := store.ClusterStoresByName[name]
	if !exists {
		clusterStore = &ClusterStore{
			Cluster:           name,
			HashWithTimestamp: make(map[string]time.Time),
			Alerts:            []*alert.EntityAlert{},
		}
		store.ClusterStoresByName[name] = clusterStore
	}
	clusterStore.parent = store
	for hash, timestamp := range clusterStore.HashWithTimestamp {
		if store.dedupDuration > 0 && now.Sub(timestamp) > store.dedupDuration {
			delete(clusterStore.HashWithTimestamp, hash)
		}
	}
	return clusterStore
}

func (clusterStore *ClusterStore) ShouldAdd(hash string, now time.Time) bool {
	timestamp, found := clusterStore.HashWithTimestamp[hash]
	if !found || clusterStore.parent.dedupDuration == 0 || now.Sub(timestamp) > clusterStore.parent.dedupDuration {
		return true
	}
	return false
}

func (clusterStore *ClusterStore) Add(entityAlert *alert.EntityAlert, hashes []string, now time.Time) {
	for _, hash := range hashes {
		clusterStore.HashWithTimestamp[hash] = now
	}
	clusterStore.Alerts = append(clusterStore.Alerts, entityAlert)
}

func (store *Store) Flush(now time.Time) error {

	store.LastRunAt = now

	if store.filePath == "" {
		return nil
	}

	content, err := json.MarshalIndent(store, "", " ")
	if err != nil {
		return fmt.Errorf("failed to serialize store to json: %v", err)
	}
	err = ioutil.WriteFile(store.filePath, content, 0777)
	if err != nil {
		return fmt.Errorf("failed to write json content to '%v': %v", store.filePath, err)
	}
	return nil
}
