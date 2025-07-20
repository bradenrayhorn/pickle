package bucket_test

import (
	"os"
	"path"
	"testing"

	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
)

func TestUploadAndDownload(t *testing.T) {
	test := newTest(t)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// upload
	err = test.bucket.UploadFile(filePath, "here.txt")
	assert.NoErr(t, err)

	// get the file
	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	upload := files[0]

	assert.Equal(t, "here.txt", upload.Path)

	// download the file
	downloadPath := path.Join(test.workingDir, "out.txt")
	err = test.bucket.DownloadFile(upload.Key, downloadPath)
	assert.NoErr(t, err)

	// check file contents
	downloaded, err := os.ReadFile(downloadPath)
	assert.NoErr(t, err)
	assert.Equal(t, "abc", string(downloaded))
}

func TestUploadAndDownloadWithNestedPaths(t *testing.T) {
	test := newTest(t)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// upload
	err = test.bucket.UploadFile(filePath, "in-folder/here.txt")
	assert.NoErr(t, err)

	// get the file
	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	upload := files[0]

	assert.Equal(t, "in-folder/here.txt", upload.Path)

	// download the file
	downloadPath := path.Join(test.workingDir, "out.txt")
	err = test.bucket.DownloadFile(upload.Key, downloadPath)
	assert.NoErr(t, err)

	// check file contents
	downloaded, err := os.ReadFile(downloadPath)
	assert.NoErr(t, err)
	assert.Equal(t, "abc", string(downloaded))
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	test := newTest(t)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// upload
	err = test.bucket.UploadFile(filePath, "here.txt")
	assert.NoErr(t, err)

	// get file
	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	upload := files[0]

	// mess with file bits
	objects := test.primaryS3.GetVersions(upload.Key)
	objects[0].Content = []byte{1, 2, 3, 4}

	// download the file
	downloadPath := path.Join(test.workingDir, "out.txt")
	err = test.bucket.DownloadFile(upload.Key, downloadPath)
	assert.ErrContains(t, err, "checksums do not match.")
}
