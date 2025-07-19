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
	slices.Reverse(objects.Versions)
	slices.Reverse(targetObjects.Versions)

	// Get only oldest version of each object
	srcObjects := map[string]*s3.ObjectMetadata{}
	dstObjects := map[string]*s3.ObjectMetadata{}

	duplicateDstObjects := []*s3.ObjectMetadata{}

	for _, object := range objects.Versions {
		meta, err := config.Client.HeadObject(object.Key, object.VersionId)
		if err != nil {
			return fmt.Errorf("get meta %s: %w", object.Key, err)
		}

		if _, ok := srcObjects[meta.PickleSHA256]; !ok {
			srcObjects[meta.PickleSHA256] = meta
		}
	}

	for _, object := range targetObjects.Versions {
		meta, err := config.Client.HeadObject(object.Key, object.VersionId)
		if err != nil {
			return fmt.Errorf("get meta %s: %w", object.Key, err)
		}

		if _, ok := dstObjects[meta.PickleSHA256]; !ok {
			duplicateDstObjects = append(duplicateDstObjects, meta)
		} else {
			dstObjects[meta.PickleSHA256] = meta
		}
	}

	toUpload := []*s3.ObjectMetadata{}
	toDelete := []*s3.ObjectMetadata{}

	// check for objects that are in src but not dst
	for _, object := range srcObjects {
		if _, ok := dstObjects[object.PickleSHA256]; !ok {
			toUpload = append(toUpload, object)
		}
	}

	// check for objects that are in dst but not src
	for _, object := range dstObjects {
		if srcMeta, ok := srcObjects[object.PickleSHA256]; ok && !srcMeta.ObjectLockRetainUntilDate.IsZero() {
			// extend lock if object is also in src AND has object lock enabled in src
			err := target.PutObjectRetention(object.Key, object.VersionID, &s3.ObjectLockRetention{
				Mode:  "COMPLIANCE",
				Until: srcMeta.ObjectLockRetainUntilDate,
			})
			if err != nil {
				return fmt.Errorf("extend lock %s: %w", object.Key, err)
			}
		} else {
			// otherwise delete it - the object is not in src
			toDelete = append(toDelete, object)
		}
	}

	// remove duplicates
	for _, object := range duplicateDstObjects {
		toDelete = append(toDelete, object)
	}

	// process uploads
	for _, object := range toUpload {
		err := target.StreamObjectTo(object.Key, object.Key, object.VersionID, config.Client, retention)
		if err != nil {
			return fmt.Errorf("failed to copy object %s: %w", object.Key, err)
		}
	}

	// process deletes
	toDeleteIdentifiers := []s3.ObjectIdentifier{}
	for _, object := range toDelete {
		toDeleteIdentifiers = append(toDeleteIdentifiers, s3.ObjectIdentifier{Key: object.Key, VersionID: object.VersionID})
	}

	if len(toDeleteIdentifiers) > 0 {
		_, err = target.DeleteObjects(toDeleteIdentifiers)
		if err != nil {
			return fmt.Errorf("delete objects: %w", err)
		}
	}

	return nil
}
