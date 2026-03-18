package management

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type apiKeyLifecycleRequest struct {
	Key       string   `json:"key"`
	Preset    string   `json:"preset"`
	ExpiresAt *string  `json:"expiresAt,omitempty"`
	Label     string   `json:"label,omitempty"`
	Models    []string `json:"models,omitempty"`
}

type apiKeyEnableRequest struct {
	Key string `json:"key"`
}

func parseLifecyclePreset(preset string, expiresAtRaw *string) (*string, *time.Time, error) {
	p := strings.TrimSpace(strings.ToLower(preset))
	if p == "" {
		v := apiKeyPresetPermanent
		return &v, nil, nil
	}
	now := time.Now().UTC()
	switch p {
	case apiKeyPreset12h:
		t := now.Add(12 * time.Hour)
		return &p, &t, nil
	case apiKeyPreset24h:
		t := now.Add(24 * time.Hour)
		return &p, &t, nil
	case apiKeyPreset7d:
		t := now.Add(7 * 24 * time.Hour)
		return &p, &t, nil
	case apiKeyPreset30d:
		t := now.Add(30 * 24 * time.Hour)
		return &p, &t, nil
	case apiKeyPresetPermanent:
		return &p, nil, nil
	case apiKeyPresetCustom:
		if expiresAtRaw == nil || strings.TrimSpace(*expiresAtRaw) == "" {
			return nil, nil, fmt.Errorf("expiresAt required for custom preset")
		}
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*expiresAtRaw))
		if err != nil {
			return nil, nil, err
		}
		u := parsed.UTC()
		return &p, &u, nil
	default:
		return nil, nil, fmt.Errorf("unsupported preset")
	}
}

func (h *Handler) ListAPIKeyLifecycle(c *gin.Context) {
	if h.keyLifecycle == nil {
		c.JSON(http.StatusOK, gin.H{"items": []apiKeyLifecycleEntry{}})
		return
	}
	items := h.keyLifecycle.list()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) SetAPIKeyLifecycle(c *gin.Context) {
	if h.keyLifecycle == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lifecycle store unavailable"})
		return
	}
	var req apiKeyLifecycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing key"})
		return
	}

	preset, expiresAt, err := parseLifecyclePreset(req.Preset, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid preset or expiresAt"})
		return
	}

	normalizedModels := normalizeModelList(req.Models)
	entry, err := h.keyLifecycle.upsert(key, func(e *apiKeyLifecycleEntry, now time.Time) {
		e.Label = strings.TrimSpace(req.Label)
		e.Preset = *preset
		e.ExpiresAt = expiresAt
		e.Models = normalizedModels
		e.Disabled = false
		e.DisabledReason = ""
		e.DisabledAt = nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !containsString(h.cfg.APIKeys, key) {
		h.cfg.APIKeys = append(h.cfg.APIKeys, key)
	}
	if !h.persist(c) {
		return
	}
	log.Debugf("api key lifecycle set for key=%s preset=%s", maskKey(key), *preset)
	_ = entry
}

func (h *Handler) DisableAPIKey(c *gin.Context) {
	if h.keyLifecycle == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lifecycle store unavailable"})
		return
	}
	var req apiKeyEnableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing key"})
		return
	}

	h.cfg.APIKeys = removeString(h.cfg.APIKeys, key)
	_, err := h.keyLifecycle.upsert(key, func(e *apiKeyLifecycleEntry, now time.Time) {
		e.Disabled = true
		e.DisabledReason = "manual"
		ts := now.UTC()
		e.DisabledAt = &ts
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !h.persist(c) {
		return
	}
	log.Debugf("api key disabled key=%s", maskKey(key))
}

func (h *Handler) EnableAPIKey(c *gin.Context) {
	if h.keyLifecycle == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lifecycle store unavailable"})
		return
	}
	var req apiKeyEnableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing key"})
		return
	}

	if !containsString(h.cfg.APIKeys, key) {
		h.cfg.APIKeys = append(h.cfg.APIKeys, key)
	}
	_, err := h.keyLifecycle.upsert(key, func(e *apiKeyLifecycleEntry, now time.Time) {
		e.Disabled = false
		e.DisabledReason = ""
		e.DisabledAt = nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !h.persist(c) {
		return
	}
	log.Debugf("api key enabled key=%s", maskKey(key))
}

func (h *Handler) DeleteAPIKeyLifecycle(c *gin.Context) {
	if h.keyLifecycle == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lifecycle store unavailable"})
		return
	}
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing key"})
		return
	}
	h.cfg.APIKeys = removeString(h.cfg.APIKeys, key)
	if err := h.keyLifecycle.delete(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !h.persist(c) {
		return
	}
	log.Debugf("api key lifecycle deleted key=%s", maskKey(key))
}

func maskKey(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func containsString(arr []string, value string) bool {
	for _, item := range arr {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}

func removeString(arr []string, value string) []string {
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if strings.TrimSpace(item) == value {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizeModelList(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}
