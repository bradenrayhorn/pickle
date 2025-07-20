package bucket_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	fakes3 "github.com/bradenrayhorn/pickle/internal/fake_s3"
	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
)

func TestMaintenance(t *testing.T) {
	test := newTest(t)
	test.setNow(time.Date(2025, time.June, 20, 0, 0, 0, 0, time.UTC))
	test.setObjectLockHours(5)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// --- Setup files for maintenance scenario ---
	err = test.bucket.UploadFile(filePath, "will-delete/a.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "will-delete/b.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "active.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "active-b.txt")
	assert.NoErr(t, err)

	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)

	fileActiveB := files[0]
	fileActive := files[1]
	fileWillDeleteA := files[2]
	fileWillDeleteB := files[3]

	assert.True(t, strings.Contains(fileWillDeleteA.Key, "will-delete/a.txt"))
	assert.True(t, strings.Contains(fileWillDeleteB.Key, "will-delete/b.txt"))
	assert.True(t, strings.Contains(fileActive.Key, "active.txt"))
	assert.True(t, strings.Contains(fileActiveB.Key, "active-b.txt"))

	// --- 1AM : do more prep
	test.setNow(test.now.Add(1 * time.Hour))

	// overwrite "active.txt" file
	data := []byte("bad data")
	crc32c, sha256 := fakes3.GetChecksums(data)
	_, err = test.client.PutObject(fileActive.Key, bytes.NewReader(data), 8, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// delete "will delete" files
	assert.NoErr(t, errors.Join(
		test.bucket.DeleteFile(fileWillDeleteA.Key),
		test.bucket.DeleteFile(fileWillDeleteB.Key),
	))

	// created orphaned checksum files
	_, err = test.client.PutObject("_pickle/checksum/orphaned-a.sha256", bytes.NewReader(data), 8, crc32c, sha256, nil)
	assert.NoErr(t, err)
	_, err = test.client.PutObject("_pickle/checksum/orphaned-b.sha256", bytes.NewReader(data), 8, crc32c, sha256, nil)
	assert.NoErr(t, err)

	test.regenerateBucket() // regenerate due to external changes

	// get file version IDs
	idWillDeleteA := test.primaryS3.GetVersionIDByFuzzyKey("will-delete/a.txt")
	idWillDeleteAChecksum := test.primaryS3.GetVersionIDByFuzzyKey(hex.EncodeToString([]byte("will-delete/a.txt")))
	idWillDeleteB := test.primaryS3.GetVersionIDByFuzzyKey("will-delete/b.txt")
	idWillDeleteBChecksum := test.primaryS3.GetVersionIDByFuzzyKey(hex.EncodeToString([]byte("will-delete/b.txt")))

	activeVersions := test.primaryS3.GetVersions(fileActive.Key)
	assert.Equal(t, 2, len(activeVersions))
	idActive := activeVersions[0].VersionID
	idActiveBadFile := activeVersions[1].VersionID
	idActiveChecksum := test.primaryS3.GetVersionIDByFuzzyKey(hex.EncodeToString([]byte("active.txt")))

	idActiveB := test.primaryS3.GetVersionIDByFuzzyKey("active-b.txt")
	idActiveBChecksum := test.primaryS3.GetVersionIDByFuzzyKey(hex.EncodeToString([]byte("active-b.txt")))
	idOrphanedAChecksum := test.primaryS3.GetVersionIDByFuzzyKey("orphaned-a.sha256")
	idOrphanedBChecksum := test.primaryS3.GetVersionIDByFuzzyKey("orphaned-b.sha256")

	// --- 2AM : first maintenance run ---
	test.setNow(test.now.Add(1 * time.Hour))
	assert.NoErr(t, test.bucket.RunMaintenance())
	// Expected changes:
	//  - Orphaned checksum is deleted
	assert.Equal(t, nil, test.primaryS3.GetByVersionID(idOrphanedAChecksum))
	assert.Equal(t, nil, test.primaryS3.GetByVersionID(idOrphanedBChecksum))
	//  - "active" bad file duplicate is deleted
	assert.Equal(t, nil, test.primaryS3.GetByVersionID(idActiveBadFile))
	//  - File locks are extended for non-marked-as-deleted files
	time7AM := time.Date(2025, time.June, 20, 7, 0, 0, 0, time.UTC)
	assert.Equal(t, time7AM, test.primaryS3.GetByVersionID(idActive).Retention.Until)
	assert.Equal(t, time7AM, test.primaryS3.GetByVersionID(idActiveChecksum).Retention.Until)
	assert.Equal(t, time7AM, test.primaryS3.GetByVersionID(idActiveB).Retention.Until)
	assert.Equal(t, time7AM, test.primaryS3.GetByVersionID(idActiveBChecksum).Retention.Until)
	//  - File locks are not extended for marked-as-deleted files
	time5AM := time.Date(2025, time.June, 20, 5, 0, 0, 0, time.UTC)
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteA).Retention.Until)
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteAChecksum).Retention.Until)
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteB).Retention.Until)
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteBChecksum).Retention.Until)
	// - Delete registry has content
	assert.True(t, len(test.primaryS3.GetVersions("_pickle/deleted")[0].Content) > 0)

	// --- 3AM : second maintenance run ---
	test.setNow(test.now.Add(1 * time.Hour))
	assert.NoErr(t, test.bucket.RunMaintenance())
	// Expected changes:
	//  - File locks are extended for non-marked-as-deleted files
	time8AM := time.Date(2025, time.June, 20, 8, 0, 0, 0, time.UTC)
	assert.Equal(t, time8AM, test.primaryS3.GetByVersionID(idActive).Retention.Until)
	assert.Equal(t, time8AM, test.primaryS3.GetByVersionID(idActiveChecksum).Retention.Until)
	assert.Equal(t, time8AM, test.primaryS3.GetByVersionID(idActiveB).Retention.Until)
	assert.Equal(t, time8AM, test.primaryS3.GetByVersionID(idActiveBChecksum).Retention.Until)
	//  - File locks are not extended for marked-as-deleted files
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteA).Retention.Until)
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteAChecksum).Retention.Until)
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteB).Retention.Until)
	assert.Equal(t, time5AM, test.primaryS3.GetByVersionID(idWillDeleteBChecksum).Retention.Until)
	// - Delete registry has content
	assert.True(t, len(test.primaryS3.GetVersions("_pickle/deleted")[0].Content) > 0)

	// --- 6AM : third maintenance run ---
	test.setNow(test.now.Add(3 * time.Hour))
	assert.NoErr(t, test.bucket.RunMaintenance())
	// Expected changes:
	//  - File locks are extended for non-marked-as-deleted files
	time11AM := time.Date(2025, time.June, 20, 11, 0, 0, 0, time.UTC)
	assert.Equal(t, time11AM, test.primaryS3.GetByVersionID(idActive).Retention.Until)
	assert.Equal(t, time11AM, test.primaryS3.GetByVersionID(idActiveChecksum).Retention.Until)
	assert.Equal(t, time11AM, test.primaryS3.GetByVersionID(idActiveB).Retention.Until)
	assert.Equal(t, time11AM, test.primaryS3.GetByVersionID(idActiveBChecksum).Retention.Until)
	//  - Marked-as-deleted files have passed their lock and are now deleted
	assert.Equal(t, nil, test.primaryS3.GetByVersionID(idWillDeleteA))
	assert.Equal(t, nil, test.primaryS3.GetByVersionID(idWillDeleteAChecksum))
	assert.Equal(t, nil, test.primaryS3.GetByVersionID(idWillDeleteB))
	assert.Equal(t, nil, test.primaryS3.GetByVersionID(idWillDeleteBChecksum))
	// - Delete registry has content
	assert.True(t, len(test.primaryS3.GetVersions("_pickle/deleted")[0].Content) > 0)

	// --- 7AM : fourth maintenance run ---
	test.setNow(test.now.Add(1 * time.Hour))
	assert.NoErr(t, test.bucket.RunMaintenance())
	// Expected changes:
	//  - File locks are extended for non-marked-as-deleted files
	time12PM := time.Date(2025, time.June, 20, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, time12PM, test.primaryS3.GetByVersionID(idActive).Retention.Until)
	assert.Equal(t, time12PM, test.primaryS3.GetByVersionID(idActiveChecksum).Retention.Until)
	assert.Equal(t, time12PM, test.primaryS3.GetByVersionID(idActiveB).Retention.Until)
	assert.Equal(t, time12PM, test.primaryS3.GetByVersionID(idActiveBChecksum).Retention.Until)
	// - Delete registry is cleaned up and left empty
	assert.True(t, len(test.primaryS3.GetVersions("_pickle/deleted")[0].Content) == 0)

}
