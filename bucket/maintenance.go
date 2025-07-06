package bucket

import (
	"errors"
	"fmt"
	"time"

	"github.com/bradenrayhorn/pickle/s3"
)

func (b *bucket) RunMaintenance() error {
	// 1. Extend object lock for all active files currently in system.
	versions, err := b.getObjectVersions()
	if err != nil {
		return err
	}

	deletedFiles, err := b.getDeletedFiles()
	if err != nil {
		return err
	}

	checksumFiles := map[string]s3.VersionInfo{}
	dataFilesToExtend := []s3.VersionInfo{}
	for _, object := range versions.Versions {
		if isDataFile(object.Key) && !deletedFiles.isDeleted(object.Key, object.VersionId) {
			dataFilesToExtend = append(dataFilesToExtend, object)
		}

		if isChecksumFile(object.Key) {
			checksumFiles[object.Key] = object
		}
	}

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

	// 2. Try to delete any files marked for deletion.
	toDelete := []s3.ObjectIdentifier{}
	for _, pair := range deletedFiles.keyVersionPairs {
		toDelete = append(toDelete, s3.ObjectIdentifier{Key: pair.key, VersionID: pair.version})
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
