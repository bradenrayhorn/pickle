package bucket

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"slices"
	"strings"
	"time"

	"github.com/bradenrayhorn/pickle/s3"
)

type deletedFiles struct {
	keys []string
}

func (df *deletedFiles) isDeleted(key string) bool {
	return slices.Contains(df.keys, key)
}

func (df *deletedFiles) append(key string) {
	df.keys = append(df.keys, key)
}

func (df *deletedFiles) remove(key string) {
	df.keys = slices.DeleteFunc(df.keys, func(k string) bool {
		return k == key
	})
}

func (df *deletedFiles) deserializeAndAddLine(line string) {
	key, err := base64.RawStdEncoding.DecodeString(line)
	if err != nil {
		return
	}

	df.append(string(key))
}

func (df *deletedFiles) serialize() string {
	lines := []string{}
	for _, key := range df.keys {
		lines = append(lines, base64.RawStdEncoding.EncodeToString([]byte(key)))
	}
	return strings.Join(lines, "\r\n")
}

var (
	deletedFilesKey = "_pickle/deleted"
)

func (b *bucket) refreshDeletedFiles() error {
	versions, err := b.getObjectVersions()
	if err != nil {
		return err
	}

	var deletedFilesVersionID string
	for _, version := range versions.Versions {
		if version.Key == deletedFilesKey && version.IsLatest {
			deletedFilesVersionID = version.VersionId
			break
		}
	}

	if deletedFilesVersionID == "" {
		// there are no deleted files
		b.cachedDeletedFiles = &deletedFiles{}
		return nil
	} else {
		// fetch and parse
		src, err := b.client.GetObject(deletedFilesKey, deletedFilesVersionID)
		if err != nil {
			return fmt.Errorf("check deleted files: %w", err)
		}
		defer func() { _ = src.Close() }()

		scanner := bufio.NewScanner(src)
		deleted := &deletedFiles{}

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				deleted.deserializeAndAddLine(line)
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("parse deleted files: %w", err)
		}

		b.cachedDeletedFiles = deleted
		return nil
	}
}

func (b *bucket) getDeletedFiles() (*deletedFiles, error) {
	if b.cachedDeletedFiles == nil {
		if err := b.refreshDeletedFiles(); err != nil {
			return nil, err
		}
		return b.cachedDeletedFiles, nil
	}

	return b.cachedDeletedFiles, nil
}

func (b *bucket) DeleteFile(key string) error {
	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return err
	}

	deletedFiles.append(key)

	return b.persistDeleteRegistry()
}

func (b *bucket) RestoreFile(key string) error {
	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return err
	}

	deletedFiles.remove(key)

	if err := b.persistDeleteRegistry(); err != nil {
		return fmt.Errorf("persist delete registry: %w", err)
	}

	retention := &s3.ObjectLockRetention{
		Mode:  "COMPLIANCE",
		Until: time.Now().Add(time.Hour * time.Duration(b.objectLockHours)),
	}

	versionID, err := b.getObjectVersionForKey(key)
	if err != nil {
		return err
	}

	if err := b.client.PutObjectRetention(key, versionID, retention); err != nil {
		return fmt.Errorf("update retention %s: %w", key, err)
	}
	return nil
}

func (b *bucket) persistDeleteRegistry() error {
	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return err
	}

	// upload new deleted registry
	serialized := []byte(deletedFiles.serialize())
	checksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	_, err = checksum.Write(serialized)
	if err != nil {
		return fmt.Errorf("write crc32 sum: %w", err)
	}

	sha256Checksum := sha256.Sum256(serialized)

	deleteResponse, err := b.client.PutObject(deletedFilesKey, bytes.NewReader(serialized), int64(len(serialized)), checksum.Sum(nil), sha256Checksum[:], nil)
	if err != nil {
		return err
	}

	// delete old deleted registries
	versions, err := b.getObjectVersions()
	if err != nil {
		return err
	}

	toDelete := []s3.ObjectIdentifier{}
	for _, version := range versions.Versions {
		if version.Key == deletedFilesKey && version.VersionId != deleteResponse.VersionID {
			toDelete = append(toDelete, s3.ObjectIdentifier{Key: version.Key, VersionID: version.VersionId})
		}
	}

	if len(toDelete) > 0 {
		_, err = b.client.DeleteObjects(toDelete)
		if err != nil {
			return fmt.Errorf("delete objects: %w", err)
		}
	}

	return nil
}
