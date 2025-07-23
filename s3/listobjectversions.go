package s3

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type VersionInfo struct {
	Key          string `xml:"Key"`
	VersionId    string `xml:"VersionId"`
	IsLatest     bool   `xml:"IsLatest"`
	LastModified string `xml:"LastModified"`
	Size         uint64 `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type DeleteMarker struct {
	Key       string `xml:"Key"`
	VersionId string `xml:"VersionId"`
	IsLatest  bool   `xml:"IsLatest"`
}

type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type ListObjectVersionsResult struct {
	Name                string         `xml:"Name"`
	Prefix              string         `xml:"Prefix"`
	KeyMarker           string         `xml:"KeyMarker"`
	VersionIdMarker     string         `xml:"VersionIdMarker"`
	MaxKeys             int            `xml:"MaxKeys"`
	IsTruncated         bool           `xml:"IsTruncated"`
	NextKeyMarker       string         `xml:"NextKeyMarker,omitempty"`
	NextVersionIdMarker string         `xml:"NextVersionIdMarker,omitempty"`
	Versions            []VersionInfo  `xml:"Version"`
	DeleteMarkers       []DeleteMarker `xml:"DeleteMarker"`
	CommonPrefixes      []CommonPrefix `xml:"CommonPrefixes"`
}

type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

func (c *Client) ListObjectVersions(prefix, keyMarker, versionIdMarker string, maxKeys int) (*ListObjectVersionsResult, error) {
	query := url.Values{}
	query.Set("versions", "")

	if prefix != "" {
		query.Set("prefix", prefix)
	}
	if keyMarker != "" {
		query.Set("key-marker", keyMarker)
	}
	if versionIdMarker != "" {
		query.Set("version-id-marker", versionIdMarker)
	}
	if maxKeys > 0 {
		query.Set("max-keys", fmt.Sprintf("%d", maxKeys))
	}

	reqURL := c.buildURL("", query)

	return withRetries(func() (*ListObjectVersionsResult, error) {
		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}

		if err := c.signV4(req, bytes.NewReader(nil)); err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("ListObjectVersions failed with status: %s, response: %s", resp.Status, string(body))

			if resp.StatusCode >= 500 {
				return nil, retriableError{err}
			} else {
				return nil, err
			}
		}

		result := &ListObjectVersionsResult{}
		if err := xml.NewDecoder(resp.Body).Decode(result); err != nil {
			return nil, fmt.Errorf("failed to parse ListObjectVersions XML: %v", err)
		}

		return result, nil
	})
}

type ListAllObjectVersionsResult struct {
	Versions      []VersionInfo
	DeleteMarkers []DeleteMarker
}

func (c *Client) ListAllObjectVersions(prefix string) (*ListAllObjectVersionsResult, error) {
	maxKeys := 1000

	keyMarker := ""
	versionIdMarker := ""

	allResult := &ListAllObjectVersionsResult{}

	for {
		result, err := c.ListObjectVersions(prefix, keyMarker, versionIdMarker, maxKeys)
		if err != nil {
			return nil, err
		}

		allResult.Versions = append(allResult.Versions, result.Versions...)
		allResult.DeleteMarkers = append(allResult.DeleteMarkers, result.DeleteMarkers...)

		if !result.IsTruncated {
			break
		}

		keyMarker = result.NextKeyMarker
		versionIdMarker = result.NextVersionIdMarker

		if keyMarker == "" && versionIdMarker == "" {
			break
		}
	}

	return allResult, nil
}
