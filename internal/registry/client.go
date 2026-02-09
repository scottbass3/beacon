package registry

import "context"

type Client interface {
	ListImages(ctx context.Context) ([]Image, error)
	ListTags(ctx context.Context, image string) ([]Tag, error)
	ListTagHistory(ctx context.Context, image, tag string) ([]HistoryEntry, error)
	DeleteTag(ctx context.Context, image, tag string) error
	RenameTag(ctx context.Context, image, from, to string) error
}
