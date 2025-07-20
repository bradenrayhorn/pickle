package bucket

import (
	"fmt"
	"time"

	"github.com/bradenrayhorn/pickle/s3"

	"filippo.io/age"
)

type Bucket struct {
	client          *s3.Client
	key             *age.X25519Identity
	objectLockHours int
	now             func() time.Time

	cachedObjectVersions *s3.ListAllObjectVersionsResult
	cachedDeletedFiles   *deletedFiles
}

type Config struct {
	Client          *s3.Client
	Key             *age.X25519Identity
	ObjectLockHours int
	NowFunc         func() time.Time
}

type BucketFile struct {
	Key          string `json:"key"`
	Path         string `json:"path"`
	IsLatest     bool   `json:"isLatest"`
	VersionID    string `json:"versionID"`
	LastModified string `json:"lastModified"`
	Size         string `json:"size"`
}

func New(config *Config) (*Bucket, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("connection is not configured")
	}

	nowFunc := func() time.Time { return time.Now() }
	if config.NowFunc != nil {
		nowFunc = config.NowFunc
	}

	return &Bucket{
		client:          config.Client,
		key:             config.Key,
		objectLockHours: config.ObjectLockHours,
		now:             nowFunc,
	}, nil
}
