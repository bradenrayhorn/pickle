package s3

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type ObjectMetadata struct {
	Key       string
	VersionID string

	PickleSHA256              string
	PickleID                  string
	ObjectLockMode            string
	ObjectLockRetainUntilDate time.Time
}

func (c *Client) HeadObject(key string, versionId string) (*ObjectMetadata, error) {
	query := url.Values{}
	if versionId != "" {
		query.Add("versionId", versionId)
	}
	reqURL := c.buildURL(key, query)

	return withRetries(func() (*ObjectMetadata, error) {
		req, err := http.NewRequest(http.MethodHead, reqURL, nil)
		if err != nil {
			return nil, err
		}

		// sign and send request
		if err := c.signV4(req, nil); err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, retriableError{err}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				body = []byte(err.Error())
			}
			err = fmt.Errorf("HeadObject failed with status: %s, response: %q", resp.Status, string(body))

			if resp.StatusCode >= 500 {
				return nil, retriableError{err}
			} else {
				return nil, err
			}
		}

		var retainUntil time.Time
		retainHeader := resp.Header.Get("x-amz-object-lock-retain-until-date")
		if retainHeader != "" {
			parsed, err := time.Parse(time.RFC3339, retainHeader)
			if err != nil {
				return nil, fmt.Errorf("parse retain time '%s' for %s: %w", retainHeader, key, err)
			}

			retainUntil = parsed
		}

		sha256 := resp.Header.Get("x-amz-meta-pickle-sha256")
		if sha256 == "" {
			return nil, fmt.Errorf("pickle-sha256 metadata missing from %s %s", key, versionId)
		}
		id := resp.Header.Get("x-amz-meta-pickle-id")
		if id == "" {
			return nil, fmt.Errorf("pickle-id metadata missing from %s %s", key, versionId)
		}

		return &ObjectMetadata{
			Key:       key,
			VersionID: resp.Header.Get("x-amz-version-id"),

			PickleID:                  id,
			PickleSHA256:              sha256,
			ObjectLockMode:            resp.Header.Get("x-amz-object-lock-mode"),
			ObjectLockRetainUntilDate: retainUntil,
		}, nil
	})
}
