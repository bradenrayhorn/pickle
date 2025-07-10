package s3

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *Client) GetObject(key string, versionId string) (io.ReadCloser, error) {
	query := url.Values{}
	if versionId != "" {
		query.Add("versionId", versionId)
	}
	reqURL := c.buildURL(key, query)

	return withRetries(func() (io.ReadCloser, error) {
		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Add("x-amz-checksum-mode", "ENABLED")

		// sign and send request
		if err := c.signV4(req, nil); err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
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

		return resp.Body, nil
	})
}
