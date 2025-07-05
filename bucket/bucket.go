package bucket

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradenrayhorn/pickle/s3"

	"filippo.io/age"
)

type bucket struct {
	client *s3.Client
	key    *age.X25519Identity
}

type Config struct {
	Client *s3.Client
	Key    *age.X25519Identity
}

type BucketFile struct {
	Name         string `json:"name"`
	IsLatest     bool   `json:"isLatest"`
	Version      string `json:"version"`
	LastModified string `json:"lastModified"`
	Size         string `json:"size"`
}

func New(config *Config) (bucket, error) {
	if config.Client == nil || config.Key == nil {
		return bucket{}, fmt.Errorf("connection is not configured")
	}

	return bucket{client: config.Client, key: config.Key}, nil
}

func (b bucket) GetFiles() ([]BucketFile, error) {
	result, err := b.client.ListAllObjectVersions("")
	if err != nil {
		return nil, fmt.Errorf("get files: %w", err)
	}

	files := []BucketFile{}
	for _, version := range result.Versions {
		// Ignore non-age files
		if !strings.HasSuffix(version.Key, ".age") {
			continue
		}

		files = append(files, BucketFile{
			Name:         strings.TrimSuffix(version.Key, ".age"),
			IsLatest:     version.IsLatest,
			Version:      version.VersionId,
			LastModified: version.LastModified,
			Size:         formatBytes(version.Size),
		})
	}

	return files, nil
}

func (b bucket) UploadFile(diskPath string, targetPath string) error {
	filename := filepath.Base(diskPath) + ".age"

	workingDir, err := os.MkdirTemp("", "marmalade-*")
	if err != nil {
		return fmt.Errorf("make working: %w", err)
	}
	defer func() { _ = os.RemoveAll(workingDir) }()

	src, err := os.Open(diskPath)
	if err != nil {
		return fmt.Errorf("open file at %s: %w", diskPath, err)
	}

	archivePath := filepath.Join(workingDir, filename)
	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create %s: %w", filename, err)
	}
	defer func() { _ = archive.Close() }()

	w, err := age.Encrypt(archive, b.key.Recipient())
	if err != nil {
		return fmt.Errorf("age encrypt: %w", err)
	}

	_, err = io.Copy(w, src)
	if err != nil {
		return fmt.Errorf("copy to age: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close encrypted file: %w", err)
	}

	stat, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("file stat: %w", err)
	}

	// get shasum
	hash := sha256.New()
	if _, err := io.Copy(hash, archive); err != nil {
		return err
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	sha256Sum := []byte(hex.EncodeToString(hash.Sum(nil)))

	err = b.client.PutObject(cleanKeyName(targetPath+".age.sha256"), bytes.NewReader(sha256Sum), int64(len(sha256Sum)), nil)
	if err != nil {
		return fmt.Errorf("upload to s3: %w", err)
	}

	err = b.client.PutObject(cleanKeyName(targetPath+".age"), archive, stat.Size(), nil)
	if err != nil {
		return fmt.Errorf("upload to s3: %w", err)
	}
	return nil
}

func (b bucket) DownloadFile(bucketKey string, bucketVersion string, diskPath string) error {
	target, err := os.Create(diskPath)
	defer func() { _ = target.Close() }()

	reader, err := b.client.GetObject(bucketKey, bucketVersion)
	defer func() {
		if reader != nil {
			_ = reader.Close()
		}
	}()
	if err != nil {
		return fmt.Errorf("get object %s: %w", bucketKey, err)
	}

	r, err := age.Decrypt(reader, b.key)
	if err != nil {
		return fmt.Errorf("decrypt %s: %w", bucketKey, err)
	}

	_, err = io.Copy(target, r)
	if err != nil {
		return fmt.Errorf("copy %s to %s: %w", bucketKey, diskPath, err)
	}

	return nil
}
