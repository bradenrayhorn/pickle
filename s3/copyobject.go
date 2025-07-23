package s3

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *Client) StreamObjectTo(toKey, key, versionID string, from *Client) error {
	_, err := withRetries(func() (*PutObjectResponse, error) {
		// First get the object
		getQuery := url.Values{}
		if versionID != "" {
			getQuery.Add("versionId", versionID)
		}
		req, err := http.NewRequest(http.MethodGet, from.buildURL(key, getQuery), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Add("x-amz-checksum-mode", "ENABLED")

		// sign and send request
		if err := from.signV4(req, nil); err != nil {
			return nil, err
		}
		resp, err := from.httpClient.Do(req)
		if err != nil {
			return nil, retriableError{err}
		}

		if resp.StatusCode != http.StatusOK {
			defer func() { _ = resp.Body.Close() }()
			body, err := io.ReadAll(resp.Body)
			if err != nil || body == nil {
				body = []byte("<nil>")
			}
			err = fmt.Errorf("GetObject failed with status: %s, response: %q", resp.Status, string(body))

			if resp.StatusCode >= 500 {
				return nil, retriableError{err}
			} else {
				return nil, err
			}
		}

		var toUpload io.Reader
		if resp.ContentLength == 0 {
			// Could not stream a nil body as Golang would add Transfer-Encoding header of "chunked"
			// if ContentLength is 0. That header value is not supported by s3 servers.
			toUpload = bytes.NewReader([]byte(""))
		} else {
			toUpload = resp.Body
		}

		// Now upload the object
		req, err = http.NewRequest(http.MethodPut, c.buildURL(toKey, nil), toUpload)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/octet-stream")
		req.ContentLength = resp.ContentLength

		retainUntil := resp.Header.Get("x-amz-object-lock-retain-until-date")
		retainMode := resp.Header.Get("x-amz-object-lock-mode")

		if retainUntil != "" && retainMode != "" {
			req.Header.Set("x-amz-object-lock-mode", retainMode)
			req.Header.Set("x-amz-object-lock-retain-until-date", retainUntil)
		}

		if c.storageClass != "" {
			req.Header.Set("x-amz-storage-class", c.storageClass)
		}

		req.Header.Set("x-amz-sdk-checksum-algorithm", "CRC32C")
		req.Header.Set("x-amz-checksum-crc32c", resp.Header.Get("x-amz-checksum-crc32c"))

		req.Header.Set("x-amz-meta-pickle-sha256", resp.Header.Get("x-amz-meta-pickle-sha256"))
		req.Header.Set("x-amz-meta-pickle-id", resp.Header.Get("x-amz-meta-pickle-id"))

		// sign and send request
		if err := c.signV4WithSum(req, resp.Header.Get("x-amz-meta-pickle-sha256")); err != nil {
			return nil, err
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, retriableError{err}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("PutObject failed with status: %s, response: %s", resp.Status, string(body))

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
