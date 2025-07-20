package fakes3

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"
)

func (s *FakeS3) handleGetObject(w http.ResponseWriter, r *http.Request, key string) {
	version := s.getObjectAndWriteHeaders(w, r, key)

	if version != nil {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(version.Content)))

		_, _ = w.Write(version.Content)
	}
}

func (s *FakeS3) handleHeadObject(w http.ResponseWriter, r *http.Request, key string) {
	_ = s.getObjectAndWriteHeaders(w, r, key)
}

func (s *FakeS3) getObjectAndWriteHeaders(w http.ResponseWriter, r *http.Request, key string) *ObjectVersion {
	s.mu.Lock()
	defer s.mu.Unlock()

	versionID := r.URL.Query().Get("versionId")

	versions, ok := s.objects[key]
	if !ok || len(versions) < 1 {
		http.Error(w, "object not found", http.StatusNotFound)
		return nil
	}

	var version *ObjectVersion
	if versionID == "" {
		// find latest
		keyVersions := slices.Collect(maps.Values(versions))
		slices.SortFunc(keyVersions, func(a *ObjectVersion, b *ObjectVersion) int {
			return strings.Compare(a.VersionID, b.VersionID)
		})
		version = keyVersions[0]
	} else {
		// find specific version
		version = versions[versionID]
	}

	if version == nil {
		http.Error(w, "version not found", http.StatusNotFound)
		return nil
	}

	// attach checksums
	if r.Header.Get("x-amz-checksum-mode") == "ENABLED" {
		if version.ChecksumType == checksumAlgorithmCRC32C {
			w.Header().Set(checksumHeaderCRC32C, version.Checksum)
		} else {
			http.Error(w, fmt.Sprintf("checksum '%s' not supported", version.ChecksumType), http.StatusInternalServerError)
			return nil
		}
	}

	// object lock
	if version.Retention != nil {
		w.Header().Set("x-amz-object-lock-mode", version.Retention.Mode)
		w.Header().Set("x-amz-object-lock-retain-until-date", version.Retention.Until.Format(time.RFC3339))
	}

	// meta
	for k, v := range version.Meta {
		w.Header().Set("x-amz-meta-"+k, v)
	}

	w.Header().Set("x-amz-version-id", version.VersionID)
	w.Header().Set("x-amz-storage-class", version.StorageClass)
	w.Header().Set("LastModified", version.LastModified.Format(time.RFC3339))

	return version
}
