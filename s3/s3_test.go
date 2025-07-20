package s3_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"io"
	"net/http"
	"testing"
	"time"

	fakes3 "github.com/bradenrayhorn/pickle/internal/fake_s3"
	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
	"github.com/bradenrayhorn/pickle/s3"
)

func getChecksums(data []byte) ([]byte, []byte) {
	sha256 := sha256.Sum256(data)
	crc32c := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	_, err := crc32c.Write(data)
	if err != nil {
		panic(err)
	}

	return crc32c.Sum(nil), sha256[:]
}

func TestCanPutAndListObjects(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")

	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// try uploading a file
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// can list the version back out
	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 0, len(result.DeleteMarkers))
	assert.Equal(t, 1, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:          "my-file.txt",
		VersionId:    v1.VersionID,
		IsLatest:     true,
		LastModified: now.Format(time.RFC3339),
		Size:         3,
		StorageClass: "STANDARD",
	}, result.Versions[0])
}

func TestDeletion(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")

	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// put a file
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// delete the specific version
	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: v1.VersionID}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	versions := sv.GetVersions("my-file.txt")
	assert.Equal(t, 0, len(versions))

	// put a file back
	v2, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// delete without version (make delete marker)
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	// check list output
	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 1, len(result.DeleteMarkers))
	assert.Equal(t, s3.DeleteMarker{
		Key:       "my-file.txt",
		VersionId: "0003",
		IsLatest:  true,
	}, result.DeleteMarkers[0])
	assert.Equal(t, 1, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:          "my-file.txt",
		VersionId:    v2.VersionID,
		IsLatest:     false,
		LastModified: now.Format(time.RFC3339),
		Size:         3,
		StorageClass: "STANDARD",
	}, result.Versions[0])

	// try deleting wrong object (silently move on)
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-filejifsoda.txt", VersionID: "blah"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	// try deleting wrong version
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "blah"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	// delete all versions
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: v2.VersionID}, {Key: "my-file.txt", VersionID: "0003"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	versions = sv.GetVersions("my-file.txt")
	assert.Equal(t, 0, len(versions))
}

func TestMultipleVersions(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")

	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// put a file twice
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)
	v2, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	assert.NotEqual(t, v1.VersionID, v2.VersionID)

	// try listing file versions
	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 0, len(result.DeleteMarkers))
	assert.Equal(t, 2, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:          "my-file.txt",
		VersionId:    v2.VersionID,
		IsLatest:     true,
		LastModified: now.Format(time.RFC3339),
		Size:         3,
		StorageClass: "STANDARD",
	}, result.Versions[0])
	assert.Equal(t, s3.VersionInfo{
		Key:          "my-file.txt",
		VersionId:    v1.VersionID,
		IsLatest:     false,
		LastModified: now.Format(time.RFC3339),
		Size:         3,
		StorageClass: "STANDARD",
	}, result.Versions[1])
}

func TestObjectRetention(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")

	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// put a file
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	version, err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// try to delete the file
	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: version.VersionID}})
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(res.Error))
	assert.Equal(t, "Object is locked", res.Error[0].Message)

	// wait two hours and try again
	sv.SetNow(now.Add(2 * time.Hour))
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: version.VersionID}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))
}

func TestObjectPutRetention(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")

	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// put a file
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	version, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// try to put retention
	err = client.PutObjectRetention("my-file.txt", version.VersionID, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// can't delete file
	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: version.VersionID}})
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(res.Error))
	assert.Equal(t, "Object is locked", res.Error[0].Message)
}

func TestObjectRetentionWithDeletionMarker(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")

	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// put a file
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	_, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// try to delete the file
	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	// try listing file versions
	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 1, len(result.DeleteMarkers))
	assert.Equal(t, s3.DeleteMarker{
		Key:       "my-file.txt",
		VersionId: "0002",
		IsLatest:  true,
	}, result.DeleteMarkers[0])
	assert.Equal(t, 1, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:          "my-file.txt",
		VersionId:    "0001",
		IsLatest:     false,
		LastModified: now.Format(time.RFC3339),
		Size:         3,
		StorageClass: "STANDARD",
	}, result.Versions[0])

	// try to delete both versions of the file
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "0001"}, {Key: "my-file.txt", VersionID: "0002"}})
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(res.Error))
	assert.Equal(t, "Object is locked", res.Error[0].Message)

	// delete marker is gone
	result, err = client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 0, len(result.DeleteMarkers))
	assert.Equal(t, 1, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:          "my-file.txt",
		VersionId:    "0001",
		IsLatest:     true,
		LastModified: now.Format(time.RFC3339),
		Size:         3,
		StorageClass: "STANDARD",
	}, result.Versions[0])

	// wait two hours and delete again
	sv.SetNow(now.Add(2 * time.Hour))
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "0001"}, {Key: "my-file.txt", VersionID: "0002"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	// everything is gone
	result, err = client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 0, len(result.DeleteMarkers))
	assert.Equal(t, 0, len(result.Versions))

	assert.Equal(t, 0, len(sv.GetVersions("my-file.txt")))
}

func TestGetObject(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// upload an object
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// try to get it back
	res, err := client.GetObject("my-file.txt", v1.VersionID)
	assert.NoErr(t, err)
	defer func() { _ = res.Close() }()

	// data should match
	reply, err := io.ReadAll(res)
	assert.NoErr(t, err)
	assert.Equal(t, hex.EncodeToString(data), hex.EncodeToString(reply))
}

func TestGetObjectDoesRetries(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// upload an object
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// the first three tries should fail, but the fourth works
	tries := 0
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		if tries < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			tries++
			return true
		}
		return false
	})

	result, err := client.GetObject("my-file.txt", v1.VersionID)
	assert.NoErr(t, err)
	_ = result.Close()

	// try another but this should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})
	_, err = client.GetObject("my-file.txt", v1.VersionID)
	assert.ErrContains(t, err, "retries exceeded")
}

func TestPutObjectDoesRetries(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// the first three tries should fail
	tries := 0
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		if tries < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			tries++
			return true
		}
		return false
	})

	// try uploading a file
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	_, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// try another upload but this should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})
	_, err = client.PutObject("my-file2.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.ErrContains(t, err, "retries exceeded")

	// check object is not uploaded
	sv.SetInterceptor(nil)
	_, err = client.HeadObject("my-file2.txt", "")
	assert.ErrContains(t, err, "Not Found")
}

func TestHeadObject(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// upload an object
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// try to get it back
	res, err := client.HeadObject("my-file.txt", v1.VersionID)
	assert.NoErr(t, err)

	assert.Equal(t, s3.ObjectMetadata{
		Key:                       "my-file.txt",
		VersionID:                 v1.VersionID,
		PickleSHA256:              hex.EncodeToString(sha256),
		ObjectLockMode:            "COMPLIANCE",
		ObjectLockRetainUntilDate: now.Add(time.Hour).Truncate(time.Second),
	}, *res)
}

func TestHeadObjectDoesRetries(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// upload an object
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// the first three tries should fail, but the fourth works
	tries := 0
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		if tries < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			tries++
			return true
		}
		return false
	})

	_, err = client.HeadObject("my-file.txt", v1.VersionID)
	assert.NoErr(t, err)

	// try another but this should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})
	_, err = client.HeadObject("my-file.txt", v1.VersionID)
	assert.ErrContains(t, err, "retries exceeded")
}

func TestCopyObject(t *testing.T) {
	src := fakes3.NewFakeS3("my-bucket")
	dst := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	src.SetNow(now)
	dst.SetNow(now)

	src.StartServer()
	dst.StartServer()
	t.Cleanup(func() { src.StopServer(); dst.StopServer() })

	srcClient := s3.NewClient(s3.Config{
		URL:       src.GetEndpoint(),
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})
	dstClient := s3.NewClient(s3.Config{
		URL:       dst.GetEndpoint(),
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// upload an object to src
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := srcClient.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// copy object into dst
	err = dstClient.StreamObjectTo("my-file.txt", "my-file.txt", v1.VersionID, srcClient)
	assert.NoErr(t, err)

	// try to get object back
	res, err := dstClient.HeadObject("my-file.txt", "0001")
	assert.NoErr(t, err)
	getResponse, err := dstClient.GetObject("my-file.txt", "0001")
	assert.NoErr(t, err)
	defer func() { _ = getResponse.Close() }()
	getBody, err := io.ReadAll(getResponse)
	assert.NoErr(t, err)

	assert.Equal(t, "abc", string(getBody))
	assert.Equal(t, s3.ObjectMetadata{
		Key:                       "my-file.txt",
		VersionID:                 v1.VersionID,
		PickleSHA256:              hex.EncodeToString(sha256),
		ObjectLockMode:            "COMPLIANCE",
		ObjectLockRetainUntilDate: now.Add(time.Hour).Truncate(time.Second),
	}, *res)
}

func TestCopyObjectDoesRetries(t *testing.T) {
	src := fakes3.NewFakeS3("my-bucket")
	dst := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	src.SetNow(now)
	dst.SetNow(now)

	src.StartServer()
	dst.StartServer()
	t.Cleanup(func() { src.StopServer(); dst.StopServer() })

	srcClient := s3.NewClient(s3.Config{
		URL:       src.GetEndpoint(),
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})
	dstClient := s3.NewClient(s3.Config{
		URL:       dst.GetEndpoint(),
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// upload an object to src
	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	v1, err := srcClient.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// the first three tries should fail, but the fourth works
	tries := 0
	src.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		if tries < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			tries++
			return true
		}
		return false
	})

	// copy object into dst
	err = dstClient.StreamObjectTo("my-file.txt", "my-file.txt", v1.VersionID, srcClient)
	assert.NoErr(t, err)

	// try another but this should fail
	src.SetInterceptor(nil)
	dst.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})
	err = dstClient.StreamObjectTo("my-file.txt", "my-file.txt", v1.VersionID, srcClient)
	assert.ErrContains(t, err, "retries exceeded")
}

func TestListObjectsDoesRetries(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	// the first three tries should fail
	tries := 0
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		if tries < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			tries++
			return true
		}
		return false
	})

	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(result.Versions))

	// try another list but this should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})
	_, err = client.ListObjectVersions("", "", "", 500)
	assert.ErrContains(t, err, "retries exceeded")
}

func TestDeleteObjectsDoesRetries(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	_, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// the first attempt to delete should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})

	_, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}})
	assert.ErrContains(t, err, "retries exceeded")

	// the first three tries should fail but eventually it will succeed
	tries := 0
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		if tries < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			tries++
			return true
		}
		return false
	})

	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "0001"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	versions := sv.GetVersions("my-file.txt")
	assert.Equal(t, 0, len(versions))
}

func TestPutObjectRetentionDoesRetries(t *testing.T) {
	sv := fakes3.NewFakeS3("my-bucket")
	now := time.Now().UTC()
	sv.SetNow(now)

	sv.StartServer()
	t.Cleanup(func() { sv.StopServer() })
	url := sv.GetEndpoint()

	client := s3.NewClient(s3.Config{
		URL:       url,
		Region:    "my-region",
		KeyID:     "keyid",
		KeySecret: "shh",
		Bucket:    "my-bucket",
		Insecure:  true,
	})

	data := []byte("abc")
	crc32c, sha256 := getChecksums(data)
	version, err := client.PutObject("my-file.txt", bytes.NewReader(data), 3, crc32c, sha256, nil)
	assert.NoErr(t, err)

	// the first attempt to put retention should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})

	err = client.PutObjectRetention("my-file.txt", version.VersionID, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: time.Now().Add(time.Hour)})
	assert.ErrContains(t, err, "retries exceeded")

	versions := sv.GetVersions("my-file.txt")
	assert.Equal(t, 1, len(versions))
	assert.Equal(t, nil, versions[0].Retention)

	// do it again, but the first three tries should fail but eventually it will succeed
	tries := 0
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		if tries < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			tries++
			return true
		}
		return false
	})

	until := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	err = client.PutObjectRetention("my-file.txt", version.VersionID, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: until})
	assert.NoErr(t, err)

	versions = sv.GetVersions("my-file.txt")
	assert.Equal(t, 1, len(versions))
	assert.Equal(t, until, versions[0].Retention.Until)
}
