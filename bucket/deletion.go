package bucket

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bradenrayhorn/pickle/s3"
)

type keyVersionPair struct {
	key     string
	version string
}

type deletedFiles struct {
	keyVersionPairs []keyVersionPair
}

func (df *deletedFiles) isDeleted(key, versionID string) bool {
	for _, pair := range df.keyVersionPairs {
		if pair.key == key && pair.version == versionID {
			return true
		}
	}
	return false
}

func (df *deletedFiles) append(key, versionID string) {
	df.keyVersionPairs = append(df.keyVersionPairs, keyVersionPair{key, versionID})
}

func (df *deletedFiles) deserializeAndAddLine(line string) {
	parts := strings.Split(strings.TrimSpace(line), "-")
	if len(parts) != 2 {
		return
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return
	}
	version, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return
	}

	df.keyVersionPairs = append(df.keyVersionPairs, keyVersionPair{
		key:     string(key),
		version: string(version),
	})
}

func (df *deletedFiles) serialize() string {
	lines := []string{}
	for _, pair := range df.keyVersionPairs {
		lines = append(lines, fmt.Sprintf("%s-%s",
			base64.RawStdEncoding.EncodeToString([]byte(pair.key)),
			base64.RawStdEncoding.EncodeToString([]byte(pair.version)),
		))
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

func (b *bucket) DeleteFile(key, versionID string) error {
	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return err
	}

	deletedFiles.append(key, versionID)

	// upload new deleted registry
	serialized := []byte(deletedFiles.serialize())
	deleteResponse, err := b.client.PutObject(deletedFilesKey, bytes.NewReader(serialized), int64(len(serialized)), nil)
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
