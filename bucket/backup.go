package bucket

import (
	"fmt"
	"slices"
	"time"

	"github.com/bradenrayhorn/pickle/s3"
)

func BackupBucket(config *Config, targetConfig s3.Config) error {
	target := s3.NewClient(targetConfig)

	objects, err := config.Client.ListAllObjectVersions("")
	if err != nil {
		return fmt.Errorf("get bucket objects: %w", err)
	}

	targetObjects, err := target.ListAllObjectVersions("")
	if err != nil {
		return fmt.Errorf("get target objects: %w", err)
	}

	retention := &s3.ObjectLockRetention{
		Mode:  "COMPLIANCE",
		Until: time.Now().Add(time.Hour * time.Duration(config.ObjectLockHours)),
	}

	// Reverse Versions so that oldest version is processed first.
	// That is important so that objects are uploaded to Destination in same order as the Source.
	slices.Reverse(objects.Versions)
	slices.Reverse(targetObjects.Versions)

	// Get only oldest version of each object
	srcObjects := map[string]s3.VersionInfo{}
	dstObjects := map[string]s3.VersionInfo{}

	for _, object := range objects.Versions {
		if !isChecksumFile(object.Key) && !isDataFile(object.Key) {
			continue
		}
		if _, ok := srcObjects[object.Key]; !ok {
			srcObjects[object.Key] = object
		}
	}

	for _, object := range targetObjects.Versions {
		if !isChecksumFile(object.Key) && !isDataFile(object.Key) {
			continue
		}
		if _, ok := dstObjects[object.Key]; !ok {
			dstObjects[object.Key] = object
		}
	}

	toUpload := []s3.VersionInfo{}
	toDelete := []s3.VersionInfo{}

	// check for objects that are in src but not dst
	for _, object := range srcObjects {
		if _, ok := dstObjects[object.Key]; !ok {
			toUpload = append(toUpload, object)
		}
	}

	// check for objects that are in dst but not src
	for _, object := range dstObjects {
		if _, ok := srcObjects[object.Key]; ok {
			err := target.PutObjectRetention(object.Key, object.VersionId, retention)
			if err != nil {
				return fmt.Errorf("extend lock %s: %w", object.Key, err)
			}
		} else {
			toDelete = append(toDelete, object)
		}
	}

	// process uploads
	for _, object := range toUpload {
		err := target.StreamObjectTo(object.Key, object.Key, object.VersionId, config.Client, retention)
		if err != nil {
			return fmt.Errorf("failed to copy object %s: %w", object.Key, err)
		}
	}

	// process deletes
	toDeleteIdentifiers := []s3.ObjectIdentifier{}
	for _, object := range toDelete {
		toDeleteIdentifiers = append(toDeleteIdentifiers, s3.ObjectIdentifier{Key: object.Key, VersionID: object.VersionId})
	}

	if len(toDeleteIdentifiers) > 0 {
		_, err = target.DeleteObjects(toDeleteIdentifiers)
		if err != nil {
			return fmt.Errorf("delete objects: %w", err)
		}
	}

	return nil
}
