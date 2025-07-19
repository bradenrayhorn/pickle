package bucket

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"filippo.io/age"
	"github.com/bradenrayhorn/pickle/s3"
	"github.com/segmentio/ksuid"
)

func (b *bucket) UploadFile(diskPath string, targetPath string) error {
	if b.key == nil {
		return fmt.Errorf("key is not configured")
	}

	workingDir, err := os.MkdirTemp("", "pickle-*")
	if err != nil {
		return fmt.Errorf("make working: %w", err)
	}
	defer func() { _ = os.RemoveAll(workingDir) }()

	src, err := os.Open(diskPath)
	if err != nil {
		return fmt.Errorf("open file at %s: %w", diskPath, err)
	}

	archivePath := filepath.Join(workingDir, "archive.age")
	archive, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create %s: %w", archivePath, err)
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
		return fmt.Errorf("close writer: %w", err)
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close encrypted file: %w", err)
	}

	stat, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("file stat: %w", err)
	}

	archive, err = os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", archivePath, err)
	}
	defer func() { _ = archive.Close() }()

	// get crc32c checksum
	checksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err := io.Copy(checksum, archive); err != nil {
		return err
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	crc32cSum := checksum.Sum(nil)

	// get shasum
	hash := sha256.New()
	if _, err := io.Copy(hash, archive); err != nil {
		return err
	}
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		return err
	}
	sha256Sum := hash.Sum(nil)
	sha256SumHex := []byte(hex.EncodeToString(sha256Sum))

	lockTime := &s3.ObjectLockRetention{
		Mode:  "COMPLIANCE",
		Until: time.Now().Add(time.Hour * time.Duration(b.objectLockHours)),
	}

	fileID := ksuid.New()
	keyName := cleanKeyName(targetPath + ".age." + fileID.String())

	_, err = b.client.PutObject(keyName, archive, stat.Size(), crc32cSum, sha256Sum, lockTime)
	if err != nil {
		return fmt.Errorf("upload to s3: %w", err)
	}

	sha256SHA256Checksum := sha256.Sum256(sha256SumHex)
	sha256CRC32Cchecksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	_, err = sha256CRC32Cchecksum.Write(sha256SumHex)
	if err != nil {
		return fmt.Errorf("crc32c checksum: %w", err)
	}
	_, err = b.client.PutObject(getChecksumPath(keyName), bytes.NewReader(sha256SumHex), int64(len(sha256SumHex)), sha256CRC32Cchecksum.Sum(nil), sha256SHA256Checksum[:], lockTime)
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
