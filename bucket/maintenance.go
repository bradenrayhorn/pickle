package bucket

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/bradenrayhorn/pickle/s3"
)

func (b *bucket) RunMaintenance() error {
	slog.Info("starting maintenance...")

	versionResult, err := b.getObjectVersions()
	if err != nil {
		return err
	}

	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return err
	}

	// get and organize files
	dataFiles := map[string]s3.VersionInfo{}
	checksumFiles := map[string]s3.VersionInfo{}
	duplicateFiles := []s3.VersionInfo{}

	potentiallyOrphanedChecksumFiles := map[string]s3.VersionInfo{}
	dataFilesToExtend := []s3.VersionInfo{}

	// Flip versions to process oldest first.
	versions := slices.Clone(versionResult.Versions)
	slices.Reverse(versions)

	for _, object := range versions {
		if isDataFile(object.Key) {
			if _, ok := dataFiles[object.Key]; ok {
				// this is a duplicate key
				duplicateFiles = append(duplicateFiles, object)
			} else {
				// it's an unprocessed key
				dataFiles[object.Key] = object

				if !deletedFiles.isDeleted(object.Key) {
					dataFilesToExtend = append(dataFilesToExtend, object)
				}
			}
		}

		if isChecksumFile(object.Key) {
			if _, ok := checksumFiles[object.Key]; ok {
				// this is a duplicate key
				duplicateFiles = append(duplicateFiles, object)
			} else {
				// it's an unprocessed key
				checksumFiles[object.Key] = object
				potentiallyOrphanedChecksumFiles[object.Key] = object
			}
		}
	}

	// calculate orphaned checksum files
	for _, object := range dataFiles {
		delete(potentiallyOrphanedChecksumFiles, getChecksumPath(object.Key))
	}
	orphanedChecksumFiles := potentiallyOrphanedChecksumFiles

	// 0. Remove any permanently deleted files from registry
	hasChangedDeleteRegistry := false
	for _, key := range deletedFiles.keys {
		if _, ok := dataFiles[key]; !ok {
			slog.Info(fmt.Sprintf("%s in registry is now removed from bucket, removing from registry", key))
			deletedFiles.remove(key)
			hasChangedDeleteRegistry = true
		}
	}
	if hasChangedDeleteRegistry {
		slog.Info("persisting new delete registry")
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
		slog.Info(fmt.Sprintf("extending retention for %s", object.Key), "versionID", object.VersionId)
		err := b.client.PutObjectRetention(object.Key, object.VersionId, retention)
		if err != nil {
			retentionErrors = append(retentionErrors, fmt.Errorf("set retention %s: %w", object.Key, err))
		}

		if checksumObject, ok := checksumFiles[getChecksumPath(object.Key)]; ok {
			slog.Info(fmt.Sprintf("extending retention for %s", checksumObject.Key), "versionID", checksumObject.VersionId)
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

	// 2. Delete any files marked for deletion, orphaned checksum files, and duplicates.
	toDelete := []s3.ObjectIdentifier{}
	for _, key := range deletedFiles.keys {
		version := dataFiles[key]

		slog.Info(fmt.Sprintf("will delete %s", key), "versionID", version.VersionId)

		toDelete = append(toDelete, s3.ObjectIdentifier{Key: version.Key, VersionID: version.VersionId})
		if checksumObject, ok := checksumFiles[getChecksumPath(key)]; ok {
			slog.Info(fmt.Sprintf("will delete %s", checksumObject.Key), "versionID", checksumObject.VersionId)
			toDelete = append(toDelete, s3.ObjectIdentifier{Key: checksumObject.Key, VersionID: checksumObject.VersionId})
		}
	}
	for _, object := range orphanedChecksumFiles {
		slog.Info(fmt.Sprintf("will delete orphaned checksum file %s", object.Key), "versionID", object.VersionId)
		toDelete = append(toDelete, s3.ObjectIdentifier{Key: object.Key, VersionID: object.VersionId})
	}
	for _, object := range duplicateFiles {
		slog.Info(fmt.Sprintf("will delete duplicate file %s", object.Key), "versionID", object.VersionId)
		toDelete = append(toDelete, s3.ObjectIdentifier{Key: object.Key, VersionID: object.VersionId})
	}

	var deleteError error
	if len(toDelete) > 0 {
		_, err = b.client.DeleteObjects(toDelete)
		if err != nil {
			deleteError = fmt.Errorf("delete objects: %w", err)
		}
	}

	slog.Info("maintenance complete")

	return errors.Join(retentionError, deleteError)
}
