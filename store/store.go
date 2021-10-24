package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"io/fs"
	"io/ioutil"
	"sort"
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
	LastRunAt         time.Time            `json:"last_run_at"`
	alerts            alert.EntityAlerts
}

func LoadOrCreate(config *config.Config, now time.Time) (*Store, error) {
	store, err := loadOrCreate(config.StoreFilePath, config.MessagesDeduplicationDuration)
	if err != nil {
		return nil, err
	}
	store.currentCluster = config.ClusterName
	clusterStore, exists := store.ClusterStoresByName[store.currentCluster]
	if exists {
		for hash, timestamp := range clusterStore.HashWithTimestamp {
			if store.dedupDuration > 0 && now.Sub(timestamp) > store.dedupDuration {
				delete(clusterStore.HashWithTimestamp, hash)
			}
		}
	} else {
		store.ClusterStoresByName[store.currentCluster] = &ClusterStore{
			Cluster:           store.currentCluster,
			HashWithTimestamp: make(map[string]time.Time),
			alerts:            []*alert.EntityAlert{},
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

func (store *Store) EntityAlerts() []*alert.EntityAlert {
	alerts := store.ClusterStoresByName[store.currentCluster].alerts
	sort.Sort(alerts)
	return alerts
}

func (store *Store) ShouldAdd(hash string, now time.Time) bool {
	clusterStore := store.ClusterStoresByName[store.currentCluster]
	timestamp, found := clusterStore.HashWithTimestamp[hash]
	if !found || store.dedupDuration == 0 || now.Sub(timestamp) > store.dedupDuration {
		return true
	}
	return false
}

func (store *Store) Add(entityAlert *alert.EntityAlert, hashes []string, now time.Time) {
	clusterStore := store.ClusterStoresByName[store.currentCluster]
	for _, hash := range hashes {
		clusterStore.HashWithTimestamp[hash] = now
	}
	clusterStore.alerts = append(clusterStore.alerts, entityAlert)
}

func (store *Store) Flush(now time.Time) error {

	store.ClusterStoresByName[store.currentCluster].LastRunAt = now

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
