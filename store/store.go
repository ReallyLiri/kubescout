package store

import (
	"KubeScout/config"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"time"
)

type Store struct {
	ClusterStoresByName map[string]*ClusterStore `json:"cluster_stores_by_name"`
	dedupDuration       time.Duration
	filePath            string
	currentCluster      string
}

type ClusterStore struct {
	Cluster           string               `json:"cluster"`
	HashWithTimestamp map[string]time.Time `json:"hash_with_timestamp"`
	messages          []string
}

func LoadOrCreate(config *config.Config) (*Store, error) {
	store, err := loadOrCreate(config.StoreFilePath, config.MessagesDeduplicationDuration)
	if err != nil {
		return nil, err
	}
	store.currentCluster = config.ClusterName
	_, exists := store.ClusterStoresByName[store.currentCluster]
	if !exists {
		store.ClusterStoresByName[store.currentCluster] = &ClusterStore{
			Cluster:           store.currentCluster,
			HashWithTimestamp: make(map[string]time.Time),
		}
	}
	return store, nil
}

func loadOrCreate(filePath string, dedupDuration time.Duration) (*Store, error) {
	store := &Store{
		ClusterStoresByName: make(map[string]*ClusterStore),
		dedupDuration:       dedupDuration,
		filePath:            filePath,
	}
	if filePath == "" {
		return store, nil
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return store, nil
		}
		return nil, fmt.Errorf("failed to read store file from '%v': %v", filePath, err)
	}
	if len(content) == 0 {
		return store, nil
	}

	err = json.Unmarshal(content, store)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize json from '%v': %v", filePath, err)
	}
	return store, nil
}

func (store *Store) RelevantMessages() []string {
	return store.ClusterStoresByName[store.currentCluster].messages
}

func (store *Store) ShouldAdd(hash string, now time.Time) bool {
	clusterStore := store.ClusterStoresByName[store.currentCluster]
	timestamp, found := clusterStore.HashWithTimestamp[hash]
	if !found || store.dedupDuration == 0 || now.Sub(timestamp) > store.dedupDuration {
		return true
	}
	return false
}

func (store *Store) Add(message string, hashes []string, now time.Time) {
	clusterStore := store.ClusterStoresByName[store.currentCluster]
	for _, hash := range hashes {
		clusterStore.HashWithTimestamp[hash] = now
	}
	clusterStore.messages = append(clusterStore.messages, message)
}

func (store *Store) Flush() error {
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
