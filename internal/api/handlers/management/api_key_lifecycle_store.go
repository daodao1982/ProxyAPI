package management

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	apiKeyPreset12h       = "12h"
	apiKeyPreset7d        = "7d"
	apiKeyPresetCustom    = "custom"
	apiKeyPresetPermanent = "permanent"
)

type apiKeyLifecycleEntry struct {
	Key            string     `json:"key"`
	Label          string     `json:"label,omitempty"`
	Preset         string     `json:"preset,omitempty"`
	ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
	Disabled       bool       `json:"disabled"`
	DisabledReason string     `json:"disabledReason,omitempty"`
	DisabledAt     *time.Time `json:"disabledAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type apiKeyLifecycleStore struct {
	mu      sync.RWMutex
	path    string
	entries map[string]*apiKeyLifecycleEntry
}

func newAPIKeyLifecycleStore(configFilePath string) *apiKeyLifecycleStore {
	storePath := ""
	if strings.TrimSpace(configFilePath) != "" {
		dir := filepath.Dir(configFilePath)
		storePath = filepath.Join(dir, ".api_key_lifecycle.json")
	}

	s := &apiKeyLifecycleStore{
		path:    storePath,
		entries: make(map[string]*apiKeyLifecycleEntry),
	}
	_ = s.load()
	return s
}

func (s *apiKeyLifecycleStore) load() error {
	if s.path == "" {
		return nil
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var payload struct {
		Entries []*apiKeyLifecycleEntry `json:"entries"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]*apiKeyLifecycleEntry)
	for _, entry := range payload.Entries {
		if entry == nil {
			continue
		}
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			continue
		}
		cloned := *entry
		s.entries[key] = &cloned
	}
	return nil
}

func (s *apiKeyLifecycleStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	entries := make([]*apiKeyLifecycleEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		cloned := *entry
		entries = append(entries, &cloned)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.Before(entries[j].CreatedAt) })

	payload := struct {
		Entries []*apiKeyLifecycleEntry `json:"entries"`
	}{Entries: entries}

	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *apiKeyLifecycleStore) list() []*apiKeyLifecycleEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]*apiKeyLifecycleEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		cloned := *entry
		entries = append(entries, &cloned)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.Before(entries[j].CreatedAt) })
	return entries
}

func (s *apiKeyLifecycleStore) upsert(key string, updater func(entry *apiKeyLifecycleEntry, now time.Time)) (*apiKeyLifecycleEntry, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("empty key")
	}
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entries[key]
	if entry == nil {
		entry = &apiKeyLifecycleEntry{Key: key, CreatedAt: now, UpdatedAt: now}
		s.entries[key] = entry
	}
	updater(entry, now)
	entry.UpdatedAt = now
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	cloned := *entry
	return &cloned, nil
}

func (s *apiKeyLifecycleStore) get(key string) (*apiKeyLifecycleEntry, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[key]
	if !ok || entry == nil {
		return nil, false
	}
	cloned := *entry
	return &cloned, true
}

func (s *apiKeyLifecycleStore) delete(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("empty key")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return s.saveLocked()
}

func (s *apiKeyLifecycleStore) disableExpired(now time.Time) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := make([]string, 0)
	for key, entry := range s.entries {
		if entry == nil || entry.Disabled || entry.ExpiresAt == nil {
			continue
		}
		if now.After(entry.ExpiresAt.UTC()) || now.Equal(entry.ExpiresAt.UTC()) {
			entry.Disabled = true
			entry.DisabledReason = "expired"
			ts := now.UTC()
			entry.DisabledAt = &ts
			entry.UpdatedAt = ts
			changed = append(changed, key)
		}
	}
	if len(changed) == 0 {
		return nil, nil
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return changed, nil
}
