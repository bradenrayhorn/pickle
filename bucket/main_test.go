package bucket_test

import (
	"testing"
	"time"

	"filippo.io/age"
	"github.com/bradenrayhorn/pickle/bucket"
	fakes3 "github.com/bradenrayhorn/pickle/internal/fake_s3"
	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
	"github.com/bradenrayhorn/pickle/s3"
)

type bucketTest struct {
	t               testing.TB
	primaryS3       *fakes3.FakeS3
	backupS3        *fakes3.FakeS3
	key             *age.X25519Identity
	objectLockHours int
	now             time.Time
	workingDir      string

	client          *s3.Client
	primaryS3Config s3.Config
	backupS3Config  s3.Config
	bucket          *bucket.Bucket
}

func (t *bucketTest) setNow(now time.Time) {
	t.now = now
	t.primaryS3.SetNow(now)
	t.backupS3.SetNow(now)
}

func (t *bucketTest) setObjectLockHours(hours int) {
	t.objectLockHours = hours
	t.regenerateBucket()
}

func (t *bucketTest) regenerateBucket() {
	bucket, err := bucket.New(&bucket.Config{
		Client:          t.client,
		Key:             t.key,
		ObjectLockHours: t.objectLockHours,
		NowFunc:         func() time.Time { return t.now },
	})
	assert.NoErr(t.t, err)
	t.bucket = bucket
}

func newTest(t testing.TB) *bucketTest {
	primaryS3 := fakes3.NewFakeS3("my-bucket")
	backupS3 := fakes3.NewFakeS3("my-bucket-backup")

	now := time.Now().UTC()
	primaryS3.SetNow(now)
	backupS3.SetNow(now)

	primaryS3.StartServer()
	backupS3.StartServer()
	t.Cleanup(func() {
		primaryS3.StopServer()
		backupS3.StopServer()
	})

	primaryS3Config := s3.Config{
		URL:       primaryS3.GetEndpoint(),
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	}
	backupS3Config := s3.Config{
		URL:       backupS3.GetEndpoint(),
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket-backup",
		Insecure:  true,
	}

	key, err := age.GenerateX25519Identity()
	assert.NoErr(t, err)

	bt := &bucketTest{
		t:               t,
		primaryS3:       primaryS3,
		backupS3:        backupS3,
		key:             key,
		objectLockHours: 0,
		now:             now,
		workingDir:      t.TempDir(),

		client:          s3.NewClient(primaryS3Config),
		primaryS3Config: primaryS3Config,
		backupS3Config:  backupS3Config,
	}
	bt.regenerateBucket()
	return bt
}
