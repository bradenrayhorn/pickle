package bucket

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/bradenrayhorn/pickle/s3"
)

func BackupBucket(sourceConfig s3.Config, targetConfig s3.Config) error {
	source := s3.NewClient(sourceConfig)
	target := s3.NewClient(targetConfig)

	slog.Info("running pickle backup...")

	objects, err := source.ListAllObjectVersions("")
	if err != nil {
		return fmt.Errorf("get bucket objects: %w", err)
	}

	targetObjects, err := target.ListAllObjectVersions("")
	if err != nil {
		return fmt.Errorf("get target objects: %w", err)
	}

	// Reverse Versions so that oldest version is processed first.
	slices.Reverse(objects.Versions)
	slices.Reverse(targetObjects.Versions)

	// Get only oldest version of each object
	srcObjects := map[string]*s3.ObjectMetadata{}
	dstObjects := map[string]*s3.ObjectMetadata{}

	duplicateDstObjects := []*s3.ObjectMetadata{}

	for _, object := range objects.Versions {
		meta, err := source.HeadObject(object.Key, object.VersionId)
		if err != nil {
			return fmt.Errorf("get meta [src] %s: %w", object.Key, err)
		}

		if _, ok := srcObjects[meta.PickleID]; !ok {
			srcObjects[meta.PickleID] = meta
		}
	}

	for _, object := range targetObjects.Versions {
		meta, err := target.HeadObject(object.Key, object.VersionId)
		if err != nil {
			return fmt.Errorf("get meta [dst] %s: %w", object.Key, err)
		}

		if _, ok := dstObjects[meta.PickleID]; !ok {
			dstObjects[meta.PickleID] = meta
		} else {
			duplicateDstObjects = append(duplicateDstObjects, meta)
			slog.Info(fmt.Sprintf("will delete duplicate object in dst at %s", object.Key), "versionID", meta.VersionID)
		}
	}

	toUpload := []*s3.ObjectMetadata{}
	toDelete := []*s3.ObjectMetadata{}

	// check for objects that are in src but not dst
	for _, object := range srcObjects {
		if _, ok := dstObjects[object.PickleID]; !ok {
			toUpload = append(toUpload, object)
			slog.Info(fmt.Sprintf("will upload %s to dst", object.Key))
		}
	}

	// check for objects that are in dst but not src
	for _, object := range dstObjects {
		if srcMeta, ok := srcObjects[object.PickleID]; ok {
			// extend lock if object is also in src AND has object lock enabled in src
			if !srcMeta.ObjectLockRetainUntilDate.IsZero() && srcMeta.ObjectLockRetainUntilDate.After(object.ObjectLockRetainUntilDate) {
				slog.Info(fmt.Sprintf("extending lock of %s in dst until %s", object.Key, srcMeta.ObjectLockRetainUntilDate.Format(time.RFC1123)), "versionID", object.VersionID)

				err := target.PutObjectRetention(object.Key, object.VersionID, &s3.ObjectLockRetention{
					Mode:  "COMPLIANCE",
					Until: srcMeta.ObjectLockRetainUntilDate,
				})
				if err != nil {
					return fmt.Errorf("extend lock %s: %w", object.Key, err)
				}
			}
		} else {
			// otherwise delete it - the object is not in src
			toDelete = append(toDelete, object)
			slog.Info(fmt.Sprintf("%s no longer in src, will delete from dst", object.Key))
		}
	}

	// remove duplicates
	for _, object := range duplicateDstObjects {
		toDelete = append(toDelete, object)
	}

	// process uploads
	for _, object := range toUpload {
		slog.Info(fmt.Sprintf("streaming %s to dst", object.Key))
		err := target.StreamObjectTo(object.Key, object.Key, object.VersionID, source)
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
		slog.Info("deleting objects...")
		_, err = target.DeleteObjects(toDeleteIdentifiers)
		if err != nil {
			return fmt.Errorf("delete objects: %w", err)
		}
	}

	slog.Info("pickle backup complete")

	return nil
}
