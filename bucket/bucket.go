package bucket

import (
	"fmt"
	"io"
	"os"

	"github.com/bradenrayhorn/pickle/s3"

	"filippo.io/age"
)

type bucket struct {
	client          *s3.Client
	key             *age.X25519Identity
	objectLockHours int

	cachedObjectVersions *s3.ListAllObjectVersionsResult
	cachedDeletedFiles   *deletedFiles
}

type Config struct {
	Client          *s3.Client
	Key             *age.X25519Identity
	ObjectLockHours int
}

type BucketFile struct {
	Name         string `json:"name"`
	IsLatest     bool   `json:"isLatest"`
	Version      string `json:"version"`
	LastModified string `json:"lastModified"`
	Size         string `json:"size"`
}

func New(config *Config) (bucket, error) {
	if config.Client == nil || config.Key == nil {
		return bucket{}, fmt.Errorf("connection is not configured")
	}

	return bucket{
		client:          config.Client,
		key:             config.Key,
		objectLockHours: config.ObjectLockHours,
	}, nil
}

func (b *bucket) DownloadFile(bucketKey string, bucketVersion string, diskPath string) error {
	target, err := os.Create(diskPath)
	defer func() { _ = target.Close() }()

	reader, err := b.client.GetObject(bucketKey, bucketVersion)
	defer func() {
		if reader != nil {
			_ = reader.Close()
		}
	}()
	if err != nil {
		return fmt.Errorf("get object %s: %w", bucketKey, err)
	}

	r, err := age.Decrypt(reader, b.key)
	if err != nil {
		return fmt.Errorf("decrypt %s: %w", bucketKey, err)
	}

	_, err = io.Copy(target, r)
	if err != nil {
		return fmt.Errorf("copy %s to %s: %w", bucketKey, diskPath, err)
	}

	return nil
}
