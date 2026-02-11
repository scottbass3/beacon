package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/scottbass3/beacon/internal/registry/history"
)

func listTagHistoryFromManifest(
	ctx context.Context,
	provider string,
	image string,
	tag string,
	getManifest func(context.Context, string, string) (history.ManifestV2, error),
	getConfig func(context.Context, string, string) (history.ConfigV2, error),
) ([]HistoryEntry, error) {
	manifest, err := getManifest(ctx, image, tag)
	if err != nil {
		return nil, err
	}
	if manifest.Config.Digest == "" {
		resolvedDigest := history.PreferredManifestDigest(manifest)
		if resolvedDigest != "" {
			manifest, err = getManifest(ctx, image, resolvedDigest)
			if err != nil {
				return nil, err
			}
		}
	}
	if manifest.Config.Digest == "" {
		return nil, fmt.Errorf("%s config digest missing for %s:%s", strings.TrimSpace(provider), image, tag)
	}
	cfg, err := getConfig(ctx, image, manifest.Config.Digest)
	if err != nil {
		return nil, err
	}
	return toHistoryEntries(history.Build(manifest, cfg)), nil
}

func toHistoryEntries(entries []history.Entry) []HistoryEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]HistoryEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, HistoryEntry{
			CreatedAt:  entry.CreatedAt,
			CreatedBy:  entry.CreatedBy,
			Comment:    entry.Comment,
			SizeBytes:  entry.SizeBytes,
			EmptyLayer: entry.EmptyLayer,
		})
	}
	return out
}
