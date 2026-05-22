package registry

import (
	"context"
	"fmt"
	"strings"
)

func listTagHistoryFromManifest(
	ctx context.Context,
	provider string,
	image string,
	tag string,
	getManifest func(context.Context, string, string) (ManifestV2, error),
	getConfig func(context.Context, string, string) (ConfigV2, error),
) ([]HistoryEntry, error) {
	manifest, err := getManifest(ctx, image, tag)
	if err != nil {
		return nil, err
	}
	if manifest.Config.Digest == "" {
		resolvedDigest := PreferredManifestDigest(manifest)
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
	return Build(manifest, cfg), nil
}
