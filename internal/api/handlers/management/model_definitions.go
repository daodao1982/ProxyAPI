package management

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
)

var staticModelChannels = []string{
	"claude",
	"gemini",
	"vertex",
	"gemini-cli",
	"aistudio",
	"codex",
	"qwen",
	"iflow",
	"kimi",
	"antigravity",
}

// GetStaticModelDefinitions returns static model metadata for a given channel.
// Channel is provided via path param (:channel) or query param (?channel=...).
func (h *Handler) GetStaticModelDefinitions(c *gin.Context) {
	channel := strings.TrimSpace(c.Param("channel"))
	if channel == "" {
		channel = strings.TrimSpace(c.Query("channel"))
	}
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel is required"})
		return
	}

	models := registry.GetStaticModelDefinitionsByChannel(channel)
	if models == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown channel", "channel": channel})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channel": strings.ToLower(strings.TrimSpace(channel)),
		"models":  models,
	})
}

// GetAllStaticModelDefinitions returns deduplicated model metadata from all known channels.
func (h *Handler) GetAllStaticModelDefinitions(c *gin.Context) {
	seen := make(map[string]struct{})
	out := make([]*registry.ModelInfo, 0, 512)

	for _, channel := range staticModelChannels {
		models := registry.GetStaticModelDefinitionsByChannel(channel)
		for _, item := range models {
			if item == nil {
				continue
			}
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}
			key := strings.ToLower(id)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, item)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ID) < strings.ToLower(out[j].ID)
	})

	c.JSON(http.StatusOK, gin.H{
		"count":  len(out),
		"models": out,
	})
}
