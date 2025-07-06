package bucket

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"filippo.io/age"
	"github.com/bradenrayhorn/pickle/s3"
)

func (b *bucket) UploadFile(diskPath string, targetPath string) error {
	filename := filepath.Base(diskPath) + ".age"

	workingDir, err := os.MkdirTemp("", "pickle-*")
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

	lockTime := &s3.ObjectLockRetention{
		Mode:  "COMPLIANCE",
		Until: time.Now().Add(time.Hour * time.Duration(b.objectLockHours)),
	}

	keyName := cleanKeyName(targetPath + ".age")

	uploadedArchive, err := b.client.PutObject(keyName, archive, stat.Size(), lockTime)
	if err != nil {
		return fmt.Errorf("upload to s3: %w", err)
	}

	_, err = b.client.PutObject(getChecksumPath(keyName, uploadedArchive.VersionID), bytes.NewReader(sha256Sum), int64(len(sha256Sum)), lockTime)
	if err != nil {
		return fmt.Errorf("upload to s3: %w", err)
	}
	return nil
}

var (
	whitespaceRegex = regexp.MustCompile(`\s+`)
	unsafeRegex     = regexp.MustCompile(`[^0-9a-zA-Z!\-_.*'()/]`)
)

func cleanKeyName(input string) string {
	result := whitespaceRegex.ReplaceAllString(input, "_")
	result = unsafeRegex.ReplaceAllString(result, "")

	return result
}
