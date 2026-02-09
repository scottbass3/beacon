package registry

import (
	"strings"
	"time"
)

type manifestV2 struct {
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		Size int64 `json:"size"`
	} `json:"layers"`
}

type configV2 struct {
	History []configHistory `json:"history"`
}

type configHistory struct {
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`
	Comment    string `json:"comment"`
	EmptyLayer bool   `json:"empty_layer"`
}

func buildHistory(manifest manifestV2, cfg configV2) []HistoryEntry {
	if len(cfg.History) == 0 {
		return nil
	}

	layerSizes := make([]int64, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		layerSizes = append(layerSizes, layer.Size)
	}

	layerIndex := 0
	entries := make([]HistoryEntry, 0, len(cfg.History))
	for _, entry := range cfg.History {
		h := HistoryEntry{
			CreatedAt:  parseDockerTime(entry.Created),
			CreatedBy:  strings.TrimSpace(entry.CreatedBy),
			Comment:    strings.TrimSpace(entry.Comment),
			SizeBytes:  -1,
			EmptyLayer: entry.EmptyLayer,
		}
		if !entry.EmptyLayer {
			if layerIndex < len(layerSizes) {
				h.SizeBytes = layerSizes[layerIndex]
				layerIndex++
			}
		}
		entries = append(entries, h)
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries
}

func parseDockerTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return time.Time{}
}
