package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/reangeline/backend_applywise/internal/core/ports/outbound"
)

type objectStorageImpl struct {
	client *awss3.Client
	bucket string
}

func NewObjectStorage(cfg aws.Config, bucket string) outbound.ObjectStorage {
	if bucket == "" {
		return nil
	}

	return &objectStorageImpl{
		client: awss3.NewFromConfig(cfg),
		bucket: bucket,
	}
}

func (s *objectStorageImpl) DeleteObjects(ctx context.Context, keys []string) error {
	objects := make([]awss3types.ObjectIdentifier, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		objects = append(objects, awss3types.ObjectIdentifier{Key: aws.String(key)})
	}

	for i := 0; i < len(objects); i += 1000 {
		end := i + 1000
		if end > len(objects) {
			end = len(objects)
		}

		out, err := s.client.DeleteObjects(ctx, &awss3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &awss3types.Delete{
				Objects: objects[i:end],
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("delete s3 objects: %w", err)
		}

		if len(out.Errors) > 0 {
			first := out.Errors[0]
			return fmt.Errorf("delete s3 objects failed for %s: %s", aws.ToString(first.Key), aws.ToString(first.Message))
		}
	}

	return nil
}