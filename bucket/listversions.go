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

	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return nil, fmt.Errorf("get deleted files: %w", err)
	}

	return versionsToBucketFiles(result.Versions, func(version s3.VersionInfo) bool {
		// Ignore deleted files
		return !deletedFiles.isDeleted(version.Key)
	}), nil
}

func (b *bucket) GetTrashedFiles() ([]BucketFile, error) {
	result, err := b.client.ListAllObjectVersions("")
	if err != nil {
		return nil, fmt.Errorf("get files: %w", err)
	}

	b.cachedObjectVersions = result

	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return nil, fmt.Errorf("get deleted files: %w", err)
	}

	return versionsToBucketFiles(result.Versions, func(version s3.VersionInfo) bool {
		// Ignore non-deleted files
		return deletedFiles.isDeleted(version.Key)
	}), nil
}

func versionsToBucketFiles(versions []s3.VersionInfo, filter func(version s3.VersionInfo) bool) []BucketFile {
	versionsToInclude := map[string]s3.VersionInfo{}
	latestIDAtPath := map[string]string{}

	for _, version := range versions {
		// Ignore non-data files
		if !isDataFile(version.Key) {
			continue
		}

		if !filter(version) {
			continue
		}

		// Versions is in newest-to-oldest order, we only want to show the oldest file because
		//   versioning is handled by the ID embedded in the key.
		versionsToInclude[version.Key] = version

		parts := strings.Split(version.Key, ".")
		if len(parts) < 3 {
			continue
		}
		id := parts[len(parts)-1]
		path := strings.TrimSuffix(version.Key, ".age."+id)

		if currentID, ok := latestIDAtPath[path]; ok {
			if id > currentID {
				latestIDAtPath[path] = id
			}
		} else {
			latestIDAtPath[path] = id
		}
	}

	files := []BucketFile{}
	for _, version := range versionsToInclude {
		parts := strings.Split(version.Key, ".")
		id := parts[len(parts)-1]
		path := strings.TrimSuffix(version.Key, ".age."+id)

		files = append(files, BucketFile{
			Key:          version.Key,
			Path:         path,
			IsLatest:     latestIDAtPath[path] == id,
			VersionID:    version.VersionId,
			LastModified: version.LastModified,
			Size:         formatBytes(version.Size),
		})
	}

	return files
}

func (b *bucket) getObjectVersionForKey(key string) (string, error) {
	versions, err := b.getObjectVersions()
	if err != nil {
		return "", err
	}

	// versions is sorted newest to oldest
	var versionID string
	for _, object := range versions.Versions {
		if object.Key == key {
			versionID = object.VersionId
		}
	}

	if versionID == "" {
		return "", fmt.Errorf("couldn't find version for object %s", key)
	}

	return versionID, nil
}
