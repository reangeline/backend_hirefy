package outbound

import "context"

// ObjectStorage defines object deletion for persisted user files.
type ObjectStorage interface {
	DeleteObjects(ctx context.Context, keys []string) error
}