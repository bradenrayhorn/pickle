package bucket_test

import (
	"bytes"
	"encoding/hex"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/bradenrayhorn/pickle/bucket"
	fakes3 "github.com/bradenrayhorn/pickle/internal/fake_s3"
	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
	"github.com/bradenrayhorn/pickle/s3"
)

func TestBackups(t *testing.T) {
	test := newTest(t)
	test.setNow(time.Date(2025, time.June, 20, 0, 0, 0, 0, time.UTC))
	test.setObjectLockHours(3)

	dstClient := s3.NewClient(test.backupS3Config)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// --- Setup files for scenario ---
	err = test.bucket.UploadFile(filePath, "deleted/a.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "active.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "active-b.txt")
	assert.NoErr(t, err)

	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)

	fileActiveB := files[0]
	fileActive := files[1]
	fileDeletedA := files[2]

	assert.True(t, strings.Contains(fileDeletedA.Key, "deleted/a.txt"))
	assert.True(t, strings.Contains(fileActive.Key, "active.txt"))
	assert.True(t, strings.Contains(fileActiveB.Key, "active-b.txt"))

	// --- 1AM : do more prep ---
	test.setNow(test.now.Add(1 * time.Hour))

	// delete "deleted" file
	assert.NoErr(t, test.bucket.DeleteFile(fileDeletedA.Key))

	// create random file in dst
	data := []byte("bad data")
	crc32c, sha256 := fakes3.GetChecksums(data)
	_, err = dstClient.PutObject("random-file.txt", bytes.NewReader(data), 8, crc32c, sha256, nil)
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(test.backupS3.GetVersions("random-file.txt")))

	// --- 2AM : first backup run ---
	test.setNow(test.now.Add(1 * time.Hour))
	assert.NoErr(t, bucket.BackupBucket(test.primaryS3Config, test.backupS3Config))
	// Expected files to be synced:
	assertSynced(t, fileActive.Key, test.primaryS3, test.backupS3)
	assertSynced(t, fileActiveB.Key, test.primaryS3, test.backupS3)
	assertSynced(t, fileDeletedA.Key, test.primaryS3, test.backupS3)
	assertSynced(t, "_pickle/deleted", test.primaryS3, test.backupS3)
	// Random file is deleted from dst:
	assert.Equal(t, 0, len(test.backupS3.GetVersions("random-file.txt")))

	// --- more setup ---
	// run maintenance - should extend object locks
	assert.NoErr(t, test.bucket.RunMaintenance())
	// create duplicate file in dst
	_, err = dstClient.PutObject(fileActive.Key, bytes.NewReader(data), 8, crc32c, sha256, nil)
	assert.NoErr(t, err)
	activeFileVersions := test.backupS3.GetVersions(fileActive.Key)
	assert.Equal(t, 2, len(activeFileVersions))
	idFileActiveGood := activeFileVersions[0].VersionID
	idFileActiveBad := activeFileVersions[1].VersionID

	assert.NotEqual(t, nil, test.backupS3.GetByVersionID(idFileActiveGood))
	assert.NotEqual(t, nil, test.backupS3.GetByVersionID(idFileActiveBad))

	// --- 3AM : second backup run ---
	test.setNow(test.now.Add(1 * time.Hour))
	assert.NoErr(t, bucket.BackupBucket(test.primaryS3Config, test.backupS3Config))
	// Expected files to be synced:
	assertSynced(t, fileActive.Key, test.primaryS3, test.backupS3)
	assertSynced(t, fileActiveB.Key, test.primaryS3, test.backupS3)
	assertSynced(t, fileDeletedA.Key, test.primaryS3, test.backupS3)
	assertSynced(t, "_pickle/deleted", test.primaryS3, test.backupS3)
	// Duplicate file is deleted from dst while original remains
	assert.NotEqual(t, nil, test.backupS3.GetByVersionID(idFileActiveGood))
	assert.Equal(t, nil, test.backupS3.GetByVersionID(idFileActiveBad))
	// fileDeletedA is still here
	assert.Equal(t, 1, len(test.backupS3.GetVersions(fileDeletedA.Key)))

	// --- 5AM : third backup run ---
	test.setNow(test.now.Add(2 * time.Hour))
	assert.NoErr(t, test.bucket.RunMaintenance()) // run maintenance in primary bucket
	assert.NoErr(t, bucket.BackupBucket(test.primaryS3Config, test.backupS3Config))
	// Expected files to be synced:
	assertSynced(t, fileActive.Key, test.primaryS3, test.backupS3)
	assertSynced(t, fileActiveB.Key, test.primaryS3, test.backupS3)
	assertSynced(t, "_pickle/deleted", test.primaryS3, test.backupS3)
	// fileDeletedA is gone now
	assert.Equal(t, 0, len(test.backupS3.GetVersions(fileDeletedA.Key)))

	// --- 7AM : fourth backup run ---
	test.setNow(test.now.Add(2 * time.Hour))
	assert.NoErr(t, test.bucket.RunMaintenance()) // run maintenance in primary bucket
	assert.NoErr(t, bucket.BackupBucket(test.primaryS3Config, test.backupS3Config))
	// Expected files to be synced:
	assertSynced(t, fileActive.Key, test.primaryS3, test.backupS3)
	assertSynced(t, fileActiveB.Key, test.primaryS3, test.backupS3)
	assertSynced(t, "_pickle/deleted", test.primaryS3, test.backupS3)
}

func assertSynced(t *testing.T, key string, from, to *fakes3.FakeS3) {
	fromVersions := from.GetVersions(key)
	toVersions := to.GetVersions(key)
	assert.Equal(t, 1, len(fromVersions))
	assert.Equal(t, 1, len(toVersions))

	src := fromVersions[0]
	dst := toVersions[0]

	assert.Equal(t, hex.EncodeToString(src.Content), hex.EncodeToString(dst.Content))
	assert.Equal(t, src.Retention, dst.Retention)
	assert.Equal(t, src.Key, dst.Key)
	assert.Equal(t, src.Checksum, dst.Checksum)
}
