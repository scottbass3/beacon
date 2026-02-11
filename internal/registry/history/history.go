package history

import (
	"strings"
	"time"
)

type ManifestV2 struct {
	MediaType string               `json:"mediaType"`
	Config    ManifestConfig       `json:"config"`
	Layers    []ManifestLayer      `json:"layers"`
	Manifests []ManifestDescriptor `json:"manifests"`
}

type ManifestConfig struct {
	Digest string `json:"digest"`
}

type ManifestLayer struct {
	Size int64 `json:"size"`
}

type ManifestDescriptor struct {
	MediaType string           `json:"mediaType"`
	Digest    string           `json:"digest"`
	Platform  ManifestPlatform `json:"platform"`
}

type ManifestPlatform struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Variant      string `json:"variant"`
}

type ConfigV2 struct {
	History []ConfigHistory `json:"history"`
}

type ConfigHistory struct {
	Created    string `json:"created"`
	CreatedBy  string `json:"created_by"`
	Comment    string `json:"comment"`
	EmptyLayer bool   `json:"empty_layer"`
}

type Entry struct {
	CreatedAt  time.Time
	CreatedBy  string
	Comment    string
	SizeBytes  int64
	EmptyLayer bool
}

func Build(manifest ManifestV2, cfg ConfigV2) []Entry {
	if len(cfg.History) == 0 {
		return nil
	}

	layerSizes := make([]int64, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		layerSizes = append(layerSizes, layer.Size)
	}

	layerIndex := 0
	entries := make([]Entry, 0, len(cfg.History))
	for _, entry := range cfg.History {
		h := Entry{
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

func PreferredManifestDigest(manifest ManifestV2) string {
	if len(manifest.Manifests) == 0 {
		return ""
	}

	bestIdx := -1
	bestScore := -1
	for i, descriptor := range manifest.Manifests {
		digest := strings.TrimSpace(descriptor.Digest)
		if digest == "" {
			continue
		}
		score := 0
		os := strings.ToLower(strings.TrimSpace(descriptor.Platform.OS))
		arch := strings.ToLower(strings.TrimSpace(descriptor.Platform.Architecture))
		variant := strings.ToLower(strings.TrimSpace(descriptor.Platform.Variant))

		if os == "linux" {
			score += 20
		}
		if arch == "amd64" || arch == "x86_64" {
			score += 10
		}
		if arch == "arm64" || arch == "aarch64" {
			score += 8
		}
		if arch == "arm" && variant != "" {
			score += 4
		}
		if descriptor.MediaType == "application/vnd.docker.distribution.manifest.v2+json" ||
			descriptor.MediaType == "application/vnd.oci.image.manifest.v1+json" {
			score += 2
		}
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	if bestIdx == -1 {
		return ""
	}
	return strings.TrimSpace(manifest.Manifests[bestIdx].Digest)
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
