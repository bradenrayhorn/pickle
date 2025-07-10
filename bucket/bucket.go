package bucket

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	Key          string `json:"key"`
	Path         string `json:"path"`
	IsLatest     bool   `json:"isLatest"`
	VersionID    string `json:"versionID"`
	LastModified string `json:"lastModified"`
	Size         string `json:"size"`
}

func New(config *Config) (bucket, error) {
	if config.Client == nil {
		return bucket{}, fmt.Errorf("connection is not configured")
	}

	return bucket{
		client:          config.Client,
		key:             config.Key,
		objectLockHours: config.ObjectLockHours,
	}, nil
}

func (b *bucket) DownloadFile(bucketKey string, diskPath string) error {
	if b.key == nil {
		return fmt.Errorf("key is not configured")
	}

	// find version id
	verisonID, err := b.getObjectVersionForKey(bucketKey)
	if err != nil {
		return err
	}

	// create working dir
	workingDir, err := os.MkdirTemp("", "pickle-*")
	if err != nil {
		return fmt.Errorf("make working: %w", err)
	}
	defer func() { _ = os.RemoveAll(workingDir) }()

	// create download file and download
	downloadPath := filepath.Join(workingDir, "download.age")
	downloadFile, err := os.Create(downloadPath)
	if err != nil {
		return fmt.Errorf("open path %s: %w", downloadPath, err)
	}
	defer func() { _ = downloadFile.Close() }()

	objectReader, err := b.client.GetObject(bucketKey, verisonID)
	defer func() {
		if objectReader != nil {
			_ = objectReader.Close()
		}
	}()
	if err != nil {
		return fmt.Errorf("get object %s: %w", bucketKey, err)
	}

	_, err = io.Copy(downloadFile, objectReader)
	if err != nil {
		return fmt.Errorf("copy s3 to disk at %s: %w", downloadPath, err)
	}

	// close download file and open for reading
	if err := downloadFile.Close(); err != nil {
		return fmt.Errorf("close %s: %w", downloadPath, err)
	}
	downloadFile, err = os.Open(downloadPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", downloadPath, err)
	}

	// compute SHA checksum if it exists
	sumSrc, err := b.client.GetObject(getChecksumPath(bucketKey), "")
	if err == nil {
		defer func() { _ = sumSrc.Close() }()
		expectedSum, err := io.ReadAll(sumSrc)
		if err != nil {
			return fmt.Errorf("read checksum: %w", err)
		}

		hash := sha256.New()
		if _, err := io.Copy(hash, downloadFile); err != nil {
			return fmt.Errorf("compute checksum: %w", err)
		}
		summedHash := hash.Sum(nil)

		// checksum was stored hex encoded, hex encode expected result before comparing
		actualSum := make([]byte, hex.EncodedLen(len(summedHash)))
		hex.Encode(actualSum, summedHash)

		if !bytes.Equal(expectedSum, actualSum) {
			return fmt.Errorf("file may have been corrupted. checksums do not match.")
		}
	}

	// reset file head
	if _, err := downloadFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek to start %s: %w", downloadPath, err)
	}

	// decrypt file to disk path
	targetFile, err := os.Create(diskPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", diskPath, err)
	}
	defer func() { _ = targetFile.Close() }()

	decryptedReader, err := age.Decrypt(downloadFile, b.key)
	if err != nil {
		return fmt.Errorf("decrypt %s: %w", bucketKey, err)
	}

	_, err = io.Copy(targetFile, decryptedReader)
	if err != nil {
		return fmt.Errorf("copy %s to %s: %w", downloadPath, diskPath, err)
	}

	return nil
}
