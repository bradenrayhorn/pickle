package bucket

import (
	"bufio"
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/bradenrayhorn/pickle/s3"
)

type deletedFiles struct {
	keyVersionPairs []string
}

func (df *deletedFiles) isDeleted(key, versionID string) bool {
	return slices.Contains(df.keyVersionPairs, fmt.Sprintf("%s-%s", key, versionID))
}

func (df *deletedFiles) append(key, versionID string) {
	df.keyVersionPairs = append(df.keyVersionPairs, fmt.Sprintf("%s-%s", key, versionID))
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
		deleted := []string{}

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				deleted = append(deleted, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("parse deleted files: %w", err)
		}

		b.cachedDeletedFiles = &deletedFiles{keyVersionPairs: deleted}
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
	fmt.Printf("appeneded deleted file %s-%s\n", key, versionID)

	// upload new deleted registry
	serialized := []byte(strings.Join(deletedFiles.keyVersionPairs, "\r\n"))
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
