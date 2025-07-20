package bucket_test

import (
	"bytes"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bradenrayhorn/pickle/bucket"
	fakes3 "github.com/bradenrayhorn/pickle/internal/fake_s3"
	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
)

func TestCanListMultipleFiles(t *testing.T) {
	test := newTest(t)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// upload a few files
	err = test.bucket.UploadFile(filePath, "here.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "here.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "nested/a.txt")
	assert.NoErr(t, err)

	err = test.bucket.UploadFile(filePath, "nested/b.txt")
	assert.NoErr(t, err)

	// get the files
	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 4, len(files))

	slices.SortFunc(files, func(a, b bucket.BucketFile) int {
		return strings.Compare(a.Key, b.Key)
	})

	assert.Equal(t, bucket.BucketFile{
		Key:          files[0].Key,
		Path:         "here.txt",
		IsLatest:     false,
		VersionID:    files[0].VersionID,
		LastModified: test.now.Format(time.RFC3339),
		Size:         files[0].Size,
	}, files[0])

	assert.Equal(t, bucket.BucketFile{
		Key:          files[1].Key,
		Path:         "here.txt",
		IsLatest:     true,
		VersionID:    files[1].VersionID,
		LastModified: test.now.Format(time.RFC3339),
		Size:         files[1].Size,
	}, files[1])

	assert.Equal(t, bucket.BucketFile{
		Key:          files[2].Key,
		Path:         "nested/a.txt",
		IsLatest:     true,
		VersionID:    files[2].VersionID,
		LastModified: test.now.Format(time.RFC3339),
		Size:         files[2].Size,
	}, files[2])

	assert.Equal(t, bucket.BucketFile{
		Key:          files[3].Key,
		Path:         "nested/b.txt",
		IsLatest:     true,
		VersionID:    files[3].VersionID,
		LastModified: test.now.Format(time.RFC3339),
		Size:         files[3].Size,
	}, files[3])
}

func TestListsOnlyLatestFileVersion(t *testing.T) {
	test := newTest(t)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// upload a few files
	err = test.bucket.UploadFile(filePath, "here.txt")
	assert.NoErr(t, err)

	// get the files
	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	upload := files[0]

	// jump ahead an hour
	test.setNow(test.now.Add(time.Hour))

	// overwrite the file
	data := []byte("bad data")
	crc32c, sha256 := fakes3.GetChecksums(data)
	_, err = test.client.PutObject(upload.Key, bytes.NewReader(data), 8, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// the original file should still be returned and downloaded
	files, err = test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	upload2 := files[0]

	assert.Equal(t, upload2.VersionID, upload.VersionID)
	assert.Equal(t, upload2.LastModified, upload.LastModified)

	downloadPath := path.Join(test.workingDir, "out.txt")
	err = test.bucket.DownloadFile(upload2.Key, downloadPath)
	assert.NoErr(t, err)

	// check file contents
	downloaded, err := os.ReadFile(downloadPath)
	assert.NoErr(t, err)
	assert.Equal(t, "abc", string(downloaded))
}
