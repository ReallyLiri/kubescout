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
	relevantMessages  []string
}

func LoadOrCreate(config *config.Config) (*Store, error) {
	if config.StoreFilePath == "" {
		return nil, nil
	}
	store, err := loadOrCreate(config.StoreFilePath, config.MessagesDeduplicationDuration)
	if err != nil {
		return nil, err
	}
	_, exists := store.ClusterStoresByName[config.ClusterName]
	if !exists {
		store.ClusterStoresByName[config.ClusterName] = &ClusterStore{
			Cluster:           config.ClusterName,
			HashWithTimestamp: make(map[string]time.Time),
		}
	}
	return store, nil
}

func loadOrCreate(filePath string, dedupDuration time.Duration) (*Store, error) {
	store := &Store{
		dedupDuration: dedupDuration,
		filePath:      filePath,
	}
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return store, nil
		}
		return nil, fmt.Errorf("failed to read store file from '%v': %v", filePath, err)
	}

	err = json.Unmarshal(content, store)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize json from '%v': %v", filePath, err)
	}
	return store, nil
}

func (store *Store) RelevantMessages() []string {
	return store.ClusterStoresByName[store.currentCluster].relevantMessages
}

func (store *Store) TryAdd(hash string, message string, now time.Time) bool {
	clusterStore := store.ClusterStoresByName[store.currentCluster]
	timestamp, found := clusterStore.HashWithTimestamp[hash]
	if !found || store.dedupDuration == 0 || now.Sub(timestamp) > store.dedupDuration {
		clusterStore.HashWithTimestamp[hash] = now
		clusterStore.relevantMessages = append(clusterStore.relevantMessages, message)
		return true
	}
	return false
}

func (store *Store) Flush() error {
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
