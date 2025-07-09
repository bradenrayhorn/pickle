package bucket

import (
	"errors"
	"fmt"
	"time"

	"github.com/bradenrayhorn/pickle/s3"
)

func (b *bucket) RunMaintenance() error {
	versions, err := b.getObjectVersions()
	if err != nil {
		return err
	}

	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return err
	}

	// get and organize files
	checksumFiles := map[string]s3.VersionInfo{}
	orphanedChecksumFiles := map[string]s3.VersionInfo{}
	duplicateChecksumFiles := []s3.VersionInfo{}

	dataFiles := []s3.VersionInfo{}
	dataFilesToExtend := []s3.VersionInfo{}
	for _, object := range versions.Versions {
		if isDataFile(object.Key) {
			dataFiles = append(dataFiles, object)

			if !deletedFiles.isDeleted(object.Key, object.VersionId) {
				dataFilesToExtend = append(dataFilesToExtend, object)
			}
		}

		if isChecksumFile(object.Key) {
			if object.IsLatest {
				checksumFiles[object.Key] = object
				orphanedChecksumFiles[object.Key] = object
			} else {
				duplicateChecksumFiles = append(duplicateChecksumFiles, object)
			}
		}
	}

	// calculate orphaned checksum files
	for _, object := range dataFiles {
		delete(orphanedChecksumFiles, getChecksumPath(object.Key, object.VersionId))
	}

	// 0. Remove any permanently deleted files from registry
	hasMadeChanges := false
	for _, pair := range deletedFiles.keyVersionPairs {
		fileStillExists := false
		for _, object := range dataFiles {
			if object.Key == pair.key && object.VersionId == pair.version {
				fileStillExists = true
				break
			}
		}

		if !fileStillExists {
			deletedFiles.remove(pair.key, pair.version)
			hasMadeChanges = true
		}
	}
	if hasMadeChanges {
		if err := b.persistDeleteRegistry(); err != nil {
			return fmt.Errorf("persist delete registry: %w", err)
		}
	}

	// 1. Extend object lock for all active files currently in system.
	retention := &s3.ObjectLockRetention{
		Mode:  "COMPLIANCE",
		Until: time.Now().Add(time.Hour * time.Duration(b.objectLockHours)),
	}
	retentionErrors := []error{}
	for _, object := range dataFilesToExtend {
		err := b.client.PutObjectRetention(object.Key, object.VersionId, retention)
		if err != nil {
			retentionErrors = append(retentionErrors, fmt.Errorf("set retention %s: %w", object.Key, err))
		}

		if checksumObject, ok := checksumFiles[getChecksumPath(object.Key, object.VersionId)]; ok {
			err := b.client.PutObjectRetention(checksumObject.Key, checksumObject.VersionId, retention)
			if err != nil {
				retentionErrors = append(retentionErrors, fmt.Errorf("set retention %s: %w", checksumObject.Key, err))
			}
		}
	}
	var retentionError error
	if len(retentionErrors) > 0 {
		retentionError = errors.Join(retentionErrors...)
	}

	// 2. Try to delete any files marked for deletion and orphaned checksum files.
	toDelete := []s3.ObjectIdentifier{}
	for _, pair := range deletedFiles.keyVersionPairs {
		toDelete = append(toDelete, s3.ObjectIdentifier{Key: pair.key, VersionID: pair.version})
		if checksumObject, ok := checksumFiles[getChecksumPath(pair.key, pair.version)]; ok {
			toDelete = append(toDelete, s3.ObjectIdentifier{Key: checksumObject.Key, VersionID: checksumObject.VersionId})
		}
	}
	for _, object := range orphanedChecksumFiles {
		toDelete = append(toDelete, s3.ObjectIdentifier{Key: object.Key, VersionID: object.VersionId})
	}
	for _, object := range duplicateChecksumFiles {
		toDelete = append(toDelete, s3.ObjectIdentifier{Key: object.Key, VersionID: object.VersionId})
	}

	var deleteError error
	if len(toDelete) > 0 {
		_, err = b.client.DeleteObjects(toDelete)
		if err != nil {
			deleteError = fmt.Errorf("delete objects: %w", err)
		}
	}

	return errors.Join(retentionError, deleteError)
}
