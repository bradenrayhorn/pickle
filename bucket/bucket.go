package bucket

import (
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

	files := make([]BucketFile, len(result.Versions))
	for i, version := range result.Versions {
		files[i] = BucketFile{
			Name:         strings.TrimSuffix(version.Key, ".age"),
			IsLatest:     version.IsLatest,
			Version:      version.VersionId,
			LastModified: "??",
			Size:         formatBytes(version.Size),
		}
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

	// TODO - checksum files
	err = b.client.PutObject(cleanKeyName(targetPath+".age"), archive, stat.Size(), nil)
	if err != nil {
		return fmt.Errorf("upload to s3: %w", err)
	}
	return nil
}
