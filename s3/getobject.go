package s3

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *Client) GetObject(key string, versionId string) (io.ReadCloser, error) {
	query := url.Values{}
	query.Add("versionId", versionId)
	reqURL := c.buildURL(key, query)

	return withRetries(func() (io.ReadCloser, error) {
		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
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

		if resp.StatusCode != http.StatusOK {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("GetObject failed with status: %s, response: %s", resp.Status, string(body))

			if resp.StatusCode >= 500 {
				return nil, retriableError{err}
			} else {
				return nil, err
			}
		}

		return resp.Body, nil
	})
}
