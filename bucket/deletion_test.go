package bucket_test

import (
	"os"
	"path"
	"testing"

	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
)

func TestCanDeleteAndRestoreFile(t *testing.T) {
	test := newTest(t)

	// create file to upload
	filePath := path.Join(test.workingDir, "file.txt")
	err := os.WriteFile(filePath, []byte("abc"), 0600)
	assert.NoErr(t, err)

	// upload a file
	err = test.bucket.UploadFile(filePath, "here.txt")
	assert.NoErr(t, err)

	// delete it
	files, err := test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	upload := files[0]

	err = test.bucket.DeleteFile(upload.Key)
	assert.NoErr(t, err)

	// the file should not be listed anymore
	files, err = test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(files))

	// but it is in the trash bin
	files, err = test.bucket.GetTrashedFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, upload.Key, files[0].Key)

	// restore the file
	err = test.bucket.RestoreFile(upload.Key)
	assert.NoErr(t, err)

	// the file should not be in the trash bin anymore
	files, err = test.bucket.GetTrashedFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(files))

	// but it is back in the main list
	files, err = test.bucket.GetFiles()
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, upload.Key, files[0].Key)
}
