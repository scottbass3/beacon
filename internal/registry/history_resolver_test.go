package registry

import (
	"context"
	"testing"

	"github.com/scottbass3/beacon/internal/registry/history"
)

func TestListTagHistoryFromManifest_ResolvesManifestList(t *testing.T) {
	calls := make([]string, 0, 2)

	getManifest := func(_ context.Context, _ string, reference string) (history.ManifestV2, error) {
		calls = append(calls, reference)
		if reference == "latest" {
			return history.ManifestV2{
				Manifests: []history.ManifestDescriptor{
					{
						Digest: "sha256:child",
						Platform: history.ManifestPlatform{
							OS:           "linux",
							Architecture: "amd64",
						},
					},
				},
			}, nil
		}
		manifest := history.ManifestV2{}
		manifest.Config.Digest = "sha256:cfg"
		manifest.Layers = []history.ManifestLayer{{Size: 42}}
		return manifest, nil
	}

	getConfig := func(_ context.Context, _ string, _ string) (history.ConfigV2, error) {
		return history.ConfigV2{
			History: []history.ConfigHistory{{CreatedBy: "RUN echo ok", EmptyLayer: false}},
		}, nil
	}

	history, err := listTagHistoryFromManifest(context.Background(), "docker hub", "library/nginx", "latest", getManifest, getConfig)
	if err != nil {
		t.Fatalf("listTagHistoryFromManifest returned error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	if len(calls) != 2 || calls[0] != "latest" || calls[1] != "sha256:child" {
		t.Fatalf("unexpected manifest resolution calls: %v", calls)
	}
}

func TestListTagHistoryFromManifest_MissingConfigDigest(t *testing.T) {
	getManifest := func(_ context.Context, _ string, _ string) (history.ManifestV2, error) {
		return history.ManifestV2{}, nil
	}
	getConfig := func(_ context.Context, _ string, _ string) (history.ConfigV2, error) {
		return history.ConfigV2{}, nil
	}

	_, err := listTagHistoryFromManifest(context.Background(), "github", "owner/image", "latest", getManifest, getConfig)
	if err == nil {
		t.Fatalf("expected missing config digest error")
	}
}
