package bucket

import (
	"fmt"
	"strings"

	"github.com/bradenrayhorn/pickle/s3"
)

func (b *bucket) getObjectVersions() (*s3.ListAllObjectVersionsResult, error) {
	if b.cachedObjectVersions != nil {
		return b.cachedObjectVersions, nil
	}

	_, err := b.GetFiles()
	if err != nil {
		return nil, err
	}

	return b.cachedObjectVersions, nil
}

func (b *bucket) GetFiles() ([]BucketFile, error) {
	result, err := b.client.ListAllObjectVersions("")
	if err != nil {
		return nil, fmt.Errorf("get files: %w", err)
	}

	b.cachedObjectVersions = result

	// Deleted files
	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return nil, fmt.Errorf("get deleted files: %w", err)
	}

	files := []BucketFile{}
	for _, version := range result.Versions {
		// Ignore non-data files
		if !isDataFile(version.Key) {
			continue
		}

		// Ignore deleted files
		if deletedFiles.isDeleted(version.Key, version.VersionId) {
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
