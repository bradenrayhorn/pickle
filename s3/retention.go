package s3

import (
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ObjectLockRetention struct {
	Mode  string // GOVERNANCE or COMPLIANCE
	Until time.Time
}

func (c *Client) PutObjectRetention(key string, versionID string, retention *ObjectLockRetention) error {
	query := url.Values{}
	query.Set("retention", "")
	query.Set("versionId", versionID)
	reqURL := c.buildURL(key, query)

	retentionXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>%s</Mode>
  <RetainUntilDate>%s</RetainUntilDate>
</Retention>`, retention.Mode, retention.Until.Format(time.RFC3339))

	_, err := withRetries(func() (any, error) {
		bodyReader := strings.NewReader(retentionXML)

		req, err := http.NewRequest(http.MethodPut, reqURL, bodyReader)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/xml")
		req.ContentLength = int64(len(retentionXML))

		// get crc32c checksum
		checksum := crc32.New(crc32.MakeTable(crc32.Castagnoli))
		if _, err := checksum.Write([]byte(retentionXML)); err != nil {
			return nil, err
		}
		crc32cChecksum := checksum.Sum(nil)

		req.Header.Set("x-amz-sdk-checksum-algorithm", "CRC32C")
		req.Header.Set("x-amz-checksum-crc32c", base64.StdEncoding.EncodeToString(crc32cChecksum))

		if err := c.signV4(req, strings.NewReader(retentionXML)); err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("PutObjectRetention failed with status: %s, response: %s", resp.Status, string(body))

			if resp.StatusCode >= 500 {
				return nil, retriableError{err}
			} else {
				return nil, err
			}
		}

		return nil, nil
	})

	return err
}
