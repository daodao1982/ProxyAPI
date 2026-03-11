package management

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/translator"
)

const defaultHealthCheckTimeout = 12 * time.Second

var healthCheckProviders = []string{
	"openai",
	"claude",
	"gemini",
	"gemini-cli",
	"vertex",
	"qwen",
	"kimi",
	"grok",
	"deepseek",
	"minimax",
	"iflow",
	"aistudio",
	"antigravity",
	"codex",
}

type modelHealthProbeRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	AuthID   string `json:"auth_id,omitempty"`
	Timeout  int    `json:"timeout_ms,omitempty"`
}

type modelHealthProbeResult struct {
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	AuthID      string    `json:"auth_id,omitempty"`
	Status      string    `json:"status"`
	HTTPStatus  int       `json:"http_status,omitempty"`
	LatencyMs   int64     `json:"latency_ms"`
	Error       string    `json:"error,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

type modelHealthChecker struct {
	mu      sync.RWMutex
	manager *coreauth.Manager
}

func newModelHealthChecker(manager *coreauth.Manager) *modelHealthChecker {
	return &modelHealthChecker{manager: manager}
}

func (c *modelHealthChecker) SetManager(manager *coreauth.Manager) {
	c.mu.Lock()
	c.manager = manager
	c.mu.Unlock()
}

func (c *modelHealthChecker) managerSnapshot() *coreauth.Manager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.manager
}

func (h *Handler) ProbeModelHealth(c *gin.Context) {
	if h == nil || h.healthChecker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "health checker unavailable"})
		return
	}

	var payload modelHealthProbeRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	provider := strings.TrimSpace(payload.Provider)
	model := strings.TrimSpace(payload.Model)
	if provider == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider and model are required"})
		return
	}

	if !isProviderAllowed(provider) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	result := h.healthChecker.probe(c.Request.Context(), payload)
	c.JSON(http.StatusOK, result)
}

func (c *modelHealthChecker) probe(ctx context.Context, payload modelHealthProbeRequest) modelHealthProbeResult {
	started := time.Now()
	result := modelHealthProbeResult{
		Provider:  payload.Provider,
		Model:     payload.Model,
		AuthID:    strings.TrimSpace(payload.AuthID),
		Status:    "error",
		CheckedAt: started.UTC(),
	}

	manager := c.managerSnapshot()
	if manager == nil {
		result.Error = "auth manager unavailable"
		return result
	}

	timeout := time.Duration(payload.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultHealthCheckTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req := cliproxyexecutor.Request{
		Model:   payload.Model,
		Format: translator.FormatOpenAI,
		Payload: []byte(`{"model":"` + payload.Model + `","messages":[{"role":"user","content":"ping"}],"max_tokens":1}`),
		Metadata: map[string]any{
			cliproxyexecutor.RequestedModelMetadataKey: payload.Model,
		},
	}

	opts := cliproxyexecutor.Options{}
	if result.AuthID != "" {
		if opts.Metadata == nil {
			opts.Metadata = map[string]any{}
		}
		opts.Metadata[cliproxyexecutor.PinnedAuthMetadataKey] = result.AuthID
	}

	resp, err := manager.Execute(ctx, []string{payload.Provider}, req, opts)
	elapsed := time.Since(started)
	result.LatencyMs = elapsed.Milliseconds()

	if err != nil {
		result.Error = err.Error()
		var statusErr interface{ StatusCode() int }
		if errors.As(err, &statusErr) {
			result.HTTPStatus = statusErr.StatusCode()
		}
		return result
	}

	if resp.Metadata != nil {
		if status, ok := resp.Metadata["status_code"].(int); ok {
			result.HTTPStatus = status
		}
	}
	if result.HTTPStatus == 0 {
		result.HTTPStatus = http.StatusOK
	}
	result.Status = "ok"
	return result
}

func isProviderAllowed(provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	for _, p := range healthCheckProviders {
		if provider == p {
			return true
		}
	}
	return false
}

func listModelHealthTargets(manager *coreauth.Manager) []modelHealthProbeRequest {
	if manager == nil {
		return nil
	}
	seen := map[string]struct{}{}
	reg := registry.GetGlobalRegistry()
	entries := manager.List()
	targets := make([]modelHealthProbeRequest, 0)

	for _, auth := range entries {
		if auth == nil || auth.Disabled || auth.Status == coreauth.StatusDisabled {
			continue
		}
		provider := strings.TrimSpace(auth.Provider)
		if provider == "" || !isProviderAllowed(provider) {
			continue
		}
		models := reg.GetModelsForClient(auth.ID)
		for _, model := range models {
			if model == nil || model.ID == "" {
				continue
			}
			key := provider + "|" + auth.ID + "|" + model.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			targets = append(targets, modelHealthProbeRequest{
				Provider: provider,
				Model:    model.ID,
				AuthID:   auth.ID,
			})
		}
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Provider == targets[j].Provider {
			return targets[i].Model < targets[j].Model
		}
		return targets[i].Provider < targets[j].Provider
	})
	return targets
}

func (h *Handler) ProbeModelHealthBatch(c *gin.Context) {
	if h == nil || h.healthChecker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "health checker unavailable"})
		return
	}

	manager := h.healthChecker.managerSnapshot()
	targets := listModelHealthTargets(manager)
	if len(targets) == 0 {
		c.JSON(http.StatusOK, gin.H{"items": []modelHealthProbeResult{}})
		return
	}

	limit := 6
	sem := make(chan struct{}, limit)
	results := make([]modelHealthProbeResult, len(targets))
	var wg sync.WaitGroup

	for i, target := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, payload modelHealthProbeRequest) {
			defer wg.Done()
			results[idx] = h.healthChecker.probe(c.Request.Context(), payload)
			<-sem
		}(i, target)
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"items": results})
}
