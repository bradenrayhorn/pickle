package s3

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
)

type PutObjectResponse struct {
	VersionID string
}

func (c *Client) PutObject(key string, data io.ReadSeeker, dataLength int64, crc32cChecksum []byte, sha256Checksum []byte, retention *ObjectLockRetention) (*PutObjectResponse, error) {
	reqURL := c.buildURL(key, nil)

	return withRetries(func() (*PutObjectResponse, error) {
		// always reset data reader at the start
		if _, err := data.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPut, reqURL, data)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/octet-stream")
		req.ContentLength = dataLength

		if retention != nil {
			req.Header.Set("x-amz-object-lock-mode", retention.Mode)
			req.Header.Set("x-amz-object-lock-retain-until-date", retention.Until.Format(time.RFC3339))
		}

		if c.storageClass != "" {
			req.Header.Set("x-amz-storage-class", c.storageClass)
		}

		req.Header.Set("x-amz-sdk-checksum-algorithm", "CRC32C")
		req.Header.Set("x-amz-checksum-crc32c", base64.StdEncoding.EncodeToString(crc32cChecksum))

		// add pickle metadata
		req.Header.Set("x-amz-meta-pickle-sha256", hex.EncodeToString(sha256Checksum))
		req.Header.Set("x-amz-meta-pickle-id", ksuid.New().String())

		// sign and send request
		if err := c.signV4WithSum(req, hex.EncodeToString(sha256Checksum)); err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, retriableError{err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("PutObject failed with status: %s, response: %s", resp.Status, string(body))

			if resp.StatusCode >= 500 {
				return nil, retriableError{err}
			} else {
				return nil, err
			}
		}

		return &PutObjectResponse{
			VersionID: resp.Header.Get("x-amz-version-id"),
		}, nil
	})
}
