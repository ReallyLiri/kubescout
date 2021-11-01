package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/reallyliri/kubescout/alert"
	"github.com/reallyliri/kubescout/config"
	"github.com/reallyliri/kubescout/dedup"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"time"
)

type Store struct {
	ClusterStoresByName map[string]*ClusterStore `json:"cluster_stores_by_name"`
	LastRunAt           time.Time                `json:"last_run_at"`
	dedupDuration       time.Duration
	filePath            string
}

type ClusterStore struct {
	parent                         *Store
	Cluster                        string                              `json:"cluster"`
	Alerts                         alert.EntityAlerts                  `json:"-"`
	MessagesWithTimestampPerEntity map[string]map[string]time.Time `json:"messages_with_timestamp_per_entity"`
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
			Cluster:                        name,
			MessagesWithTimestampPerEntity: make(map[string]map[string]time.Time),
			Alerts:                         []*alert.EntityAlert{},
		}
		store.ClusterStoresByName[name] = clusterStore
	}
	clusterStore.parent = store
	for entityName, messagesByTimestamp := range clusterStore.MessagesWithTimestampPerEntity {
		for message, timestamp := range messagesByTimestamp {
			if store.dedupDuration > 0 && now.Sub(timestamp) > store.dedupDuration {
				delete(messagesByTimestamp, message)
			}
		}
		if len(messagesByTimestamp) == 0 {
			delete(clusterStore.MessagesWithTimestampPerEntity, entityName)
		}
	}
	return clusterStore
}

func tryMatch(messagesByTimestamp map[string]time.Time, candidate string) (match string) {
	if _, found := messagesByTimestamp[candidate]; found {
		return candidate
	}

	const similarityThreshold = 0.85
	for stored := range messagesByTimestamp {
		if dedup.AreSimilar(stored, candidate, similarityThreshold) {
			return stored
		}
	}
	return ""
}

func (clusterStore *ClusterStore) TryAdd(entityName EntityName, message string, now time.Time) bool {
	message = dedup.NormalizeTemporal(message)
	truncMessage := message
	if len(message) > 50 {
		truncMessage = message[:50] + "..."
	}

	messagesByTimestamp, found := clusterStore.MessagesWithTimestampPerEntity[entityName.String()]
	if !found {
		log.Tracef("no match was found for message '%v' for entity %v - adding it", truncMessage, entityName)
		messagesByTimestamp = map[string]time.Time{
			message: now,
		}
		clusterStore.MessagesWithTimestampPerEntity[entityName.String()] = messagesByTimestamp
		return true
	}

	match := tryMatch(messagesByTimestamp, message)
	if match != "" {
		timestamp := messagesByTimestamp[match]
		if clusterStore.parent.dedupDuration > 0 && now.Sub(timestamp) <= clusterStore.parent.dedupDuration {
			log.Tracef("match was found for message '%v' for entity %v and its timestamp is in dedup grace time - skipping", truncMessage, entityName)
			return false
		}
		log.Tracef("match was found for message '%v' for entity %v but its timestamp is out of dedup grace time - adding it", truncMessage, entityName)
		messagesByTimestamp[message] = now
		return true
	}

	log.Tracef("no match was found for message '%v' for entity %v - adding it", truncMessage, entityName)
	messagesByTimestamp[message] = now
	return true
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
