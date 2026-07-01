package gcp

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

// HMACKey is an S3-interop credential pair for GCS.
type HMACKey struct {
	AccessID string
	Secret   string
}

// CreateBucket creates a GCS bucket, treating "already exists" as success.
func CreateBucket(ctx context.Context, project, name, location string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	err = client.Bucket(name).Create(ctx, project, &storage.BucketAttrs{Location: location})
	if err != nil && !isConflict(err) {
		return fmt.Errorf("create bucket %s: %w", name, err)
	}
	return nil
}

// CreateHMACKey mints an HMAC key for the service account (needs objectAdmin
// on the buckets). The secret is only returned at creation time.
func CreateHMACKey(ctx context.Context, project, serviceAccount string) (HMACKey, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return HMACKey{}, err
	}
	defer client.Close()

	key, err := client.CreateHMACKey(ctx, project, serviceAccount)
	if err != nil {
		return HMACKey{}, fmt.Errorf("create HMAC key: %w", err)
	}
	return HMACKey{AccessID: key.AccessID, Secret: key.Secret}, nil
}

// DeleteBucket empties and deletes a bucket, ignoring "not found".
func DeleteBucket(ctx context.Context, name string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket(name)
	it := bucket.Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		if err := bucket.Object(attrs.Name).Delete(ctx); err != nil {
			return err
		}
	}
	if err := bucket.Delete(ctx); err != nil && !errors.Is(err, storage.ErrBucketNotExist) {
		return fmt.Errorf("delete bucket %s: %w", name, err)
	}
	return nil
}

func isConflict(err error) bool {
	var gerr *googleapi.Error
	return errors.As(err, &gerr) && gerr.Code == 409
}
