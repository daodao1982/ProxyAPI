package management

import (
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
)

func (h *Handler) startAPIKeyLifecycleWorker() {
	if h == nil || h.keyLifecycle == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			h.applyExpiredAPIKeys()
		}
	}()
}

func (h *Handler) applyExpiredAPIKeys() {
	if h == nil || h.keyLifecycle == nil || h.cfg == nil {
		return
	}
	now := time.Now().UTC()
	expiredKeys, err := h.keyLifecycle.disableExpired(now)
	if err != nil {
		logging.Warnf("api key lifecycle: disableExpired failed: %v", err)
		return
	}
	if len(expiredKeys) == 0 {
		return
	}

	changed := false
	for _, key := range expiredKeys {
		before := len(h.cfg.APIKeys)
		h.cfg.APIKeys = removeString(h.cfg.APIKeys, key)
		if len(h.cfg.APIKeys) != before {
			changed = true
		}
	}
	if !changed {
		return
	}
	if err := h.persistNoResponse(); err != nil {
		logging.Errorf("api key lifecycle: persist failed: %v", err)
		return
	}
	logging.Infof("api key lifecycle: disabled %d expired keys", len(expiredKeys))
}
