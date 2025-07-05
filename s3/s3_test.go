package s3_test

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	fakes3 "github.com/bradenrayhorn/pickle/internal/fake_s3"
	"github.com/bradenrayhorn/pickle/internal/testutils/assert"
	"github.com/bradenrayhorn/pickle/s3"
)

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
	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)

	versions := sv.GetVersions("my-file.txt")
	assert.Equal(t, 1, len(versions))

	assert.Equal(t, versions[0], &fakes3.ObjectVersion{
		Key:          "my-file.txt",
		VersionID:    "v1",
		Content:      []byte("abc"),
		LastModified: now,
		StorageClass: "STANDARD",
		DeleteMarker: false,
		Retention:    nil,
	})

	// can list the version back out
	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 0, len(result.DeleteMarkers))
	assert.Equal(t, 1, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:       "my-file.txt",
		VersionId: "v1",
		IsLatest:  true,
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
	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)

	// delete the specific version
	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	versions := sv.GetVersions("my-file.txt")
	assert.Equal(t, 0, len(versions))

	// put a file back
	err = client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)

	// delete without version (make delete marker)
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt"}})
	assert.NoErr(t, err)
	assert.Equal(t, 0, len(res.Error))

	versions = sv.GetVersions("my-file.txt")
	assert.Equal(t, 2, len(versions))
	assert.Equal(t, versions[0], &fakes3.ObjectVersion{
		Key:          "my-file.txt",
		VersionID:    "v2",
		Content:      []byte("abc"),
		LastModified: now,
		StorageClass: "STANDARD",
		DeleteMarker: false,
		Retention:    nil,
	})
	assert.Equal(t, versions[1], &fakes3.ObjectVersion{
		Key:          "my-file.txt",
		VersionID:    "v3",
		LastModified: now,
		DeleteMarker: true,
		Retention:    nil,
	})

	// check list output
	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 1, len(result.DeleteMarkers))
	assert.Equal(t, s3.DeleteMarker{
		Key:       "my-file.txt",
		VersionId: "v3",
		IsLatest:  true,
	}, result.DeleteMarkers[0])
	assert.Equal(t, 1, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:       "my-file.txt",
		VersionId: "v2",
		IsLatest:  false,
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
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v2"}, {Key: "my-file.txt", VersionID: "v3"}})
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
	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)
	err = client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)

	// try listing file versions
	result, err := client.ListObjectVersions("", "", "", 500)
	assert.NoErr(t, err)

	assert.Equal(t, false, result.IsTruncated)
	assert.Equal(t, 0, len(result.DeleteMarkers))
	assert.Equal(t, 2, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:       "my-file.txt",
		VersionId: "v2",
		IsLatest:  true,
	}, result.Versions[0])
	assert.Equal(t, s3.VersionInfo{
		Key:       "my-file.txt",
		VersionId: "v1",
		IsLatest:  false,
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
	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// try to delete the file
	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}})
	assert.NoErr(t, err)
	assert.Equal(t, 1, len(res.Error))
	assert.Equal(t, "Object is locked", res.Error[0].Message)

	// wait two hours and try again
	sv.SetNow(now.Add(2 * time.Hour))
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}})
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
	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)

	// try to put retention
	err = client.PutObjectRetention("my-file.txt", &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
	assert.NoErr(t, err)

	// can't delete file
	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}})
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
	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: now.Add(time.Hour)})
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
		VersionId: "v2",
		IsLatest:  true,
	}, result.DeleteMarkers[0])
	assert.Equal(t, 1, len(result.Versions))
	assert.Equal(t, s3.VersionInfo{
		Key:       "my-file.txt",
		VersionId: "v1",
		IsLatest:  false,
	}, result.Versions[0])

	// try to delete both versions of the file
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}, {Key: "my-file.txt", VersionID: "v2"}})
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
		Key:       "my-file.txt",
		VersionId: "v1",
		IsLatest:  true,
	}, result.Versions[0])

	// wait two hours and delete again
	sv.SetNow(now.Add(2 * time.Hour))
	res, err = client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}, {Key: "my-file.txt", VersionID: "v2"}})
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
	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)

	versions := sv.GetVersions("my-file.txt")
	assert.Equal(t, 1, len(versions))

	assert.Equal(t, versions[0], &fakes3.ObjectVersion{
		Key:          "my-file.txt",
		VersionID:    "v1",
		Content:      []byte("abc"),
		LastModified: now,
		StorageClass: "STANDARD",
		DeleteMarker: false,
		Retention:    nil,
	})

	// try another upload but this should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})
	err = client.PutObject("my-file2.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.ErrContains(t, err, "retries exceeded")

	versions = sv.GetVersions("my-file2.txt")
	assert.Equal(t, 0, len(versions))
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

	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
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

	res, err := client.DeleteObjects([]s3.ObjectIdentifier{{Key: "my-file.txt", VersionID: "v1"}})
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

	err := client.PutObject("my-file.txt", bytes.NewReader([]byte("abc")), 3, nil)
	assert.NoErr(t, err)

	// the first attempt to put retention should fail
	sv.SetInterceptor(func(r *http.Request, w http.ResponseWriter) bool {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	})

	err = client.PutObjectRetention("my-file.txt", &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: time.Now().Add(time.Hour)})
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
	err = client.PutObjectRetention("my-file.txt", &s3.ObjectLockRetention{Mode: "COMPLIANCE", Until: until})
	assert.NoErr(t, err)

	versions = sv.GetVersions("my-file.txt")
	assert.Equal(t, 1, len(versions))
	assert.Equal(t, until, versions[0].Retention.Until)
}
