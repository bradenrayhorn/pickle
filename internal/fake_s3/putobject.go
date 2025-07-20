package fakes3

import (
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	checksumAlgorithmCRC32C = "CRC32C"
	checksumHeaderCRC32C    = "x-amz-checksum-crc32c"
)

func (s *FakeS3) handlePutObject(w http.ResponseWriter, r *http.Request, key string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading body: %v", err), http.StatusBadRequest)
		return
	}

	obj := &ObjectVersion{
		Key:          key,
		Content:      body,
		LastModified: s.now,
		StorageClass: "STANDARD",
		Meta:         map[string]string{},
	}

	// storage class
	if sc := r.Header.Get("x-amz-storage-class"); sc != "" {
		obj.StorageClass = sc
	}

	// checksum
	algorithm := r.Header.Get("x-amz-sdk-checksum-algorithm")
	if algorithm == checksumAlgorithmCRC32C {
		proposedChecksum := r.Header.Get(checksumHeaderCRC32C)

		crc := crc32.New(crc32.MakeTable(crc32.Castagnoli))
		_, err = crc.Write(body)
		if err != nil {
			panic(err)
		}

		expectedChecksum := base64.StdEncoding.EncodeToString(crc.Sum(nil))

		if proposedChecksum != expectedChecksum {
			http.Error(w, fmt.Sprintf("Proposed checksum '%s' does not equal expected '%s'", proposedChecksum, expectedChecksum), http.StatusBadRequest)
			return
		}

		obj.Checksum = proposedChecksum
		obj.ChecksumType = algorithm
	} else {
		http.Error(w, "Missing checksum.", http.StatusBadRequest)
		return
	}

	// meta
	for k, v := range r.Header {
		k = strings.ToLower(k)
		if strings.HasPrefix(k, "x-amz-meta-") && len(v) == 1 {
			obj.Meta[strings.TrimPrefix(k, "x-amz-meta-")] = v[0]
		}
	}

	// object retention
	lockMode := r.Header.Get("x-amz-object-lock-mode")
	lockDate := r.Header.Get("x-amz-object-lock-retain-until-date")
	if lockMode != "" && lockDate != "" {
		retainUntil, err := time.Parse(time.RFC3339, lockDate)
		if err == nil {
			obj.Retention = &ObjectLockRetention{
				Mode:  lockMode,
				Until: retainUntil,
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// generate version id
	versionID := s.generateVersionID()
	obj.VersionID = versionID
	w.Header().Set("x-amz-version-id", versionID)

	// save object
	if _, exists := s.objects[key]; !exists {
		s.objects[key] = make(map[string]*ObjectVersion)
	}

	s.objects[key][versionID] = obj

	w.WriteHeader(http.StatusOK)
}
