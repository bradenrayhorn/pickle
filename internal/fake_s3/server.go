package fakes3

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	ownerID   = "1234"
	ownerName = "One Two Three Four"
)

type ObjectVersion struct {
	Key          string
	VersionID    string
	Content      []byte
	LastModified time.Time
	StorageClass string
	DeleteMarker bool
	ChecksumType string
	Checksum     string
	Retention    *ObjectLockRetention
	Meta         map[string]string
}

type ObjectLockRetention struct {
	Mode  string // GOVERNANCE or COMPLIANCE
	Until time.Time
}

type FakeS3 struct {
	mu            sync.RWMutex
	server        *httptest.Server
	bucket        string
	objects       map[string]map[string]*ObjectVersion // map[key]map[versionID]*ObjectVersion
	nextVersionID int
	now           time.Time

	interceptor func(r *http.Request, w http.ResponseWriter) bool
}

func NewFakeS3(bucket string) *FakeS3 {
	return &FakeS3{
		objects: make(map[string]map[string]*ObjectVersion),
		bucket:  bucket,
		now:     time.Now().UTC(),
	}
}

func (s *FakeS3) StartServer() {
	s.server = httptest.NewServer(http.HandlerFunc(s.handleRequest))
}

func (s *FakeS3) StopServer() {
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
}

func (s *FakeS3) SetNow(time time.Time) {
	s.now = time.UTC()
}

func (s *FakeS3) SetInterceptor(i func(r *http.Request, w http.ResponseWriter) bool) {
	s.interceptor = i
}

func (s *FakeS3) GetEndpoint() string {
	if s.server == nil {
		return ""
	}
	u, _ := url.Parse(s.server.URL)
	return u.Host
}

func (s *FakeS3) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.objects = make(map[string]map[string]*ObjectVersion)
}

func (s *FakeS3) GetVersions(key string) []*ObjectVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if versions, ok := s.objects[key]; ok {
		v := slices.Collect(maps.Values(versions))

		slices.SortFunc(v, func(a, b *ObjectVersion) int {
			return strings.Compare(a.VersionID, b.VersionID)
		})

		return v
	}

	return []*ObjectVersion{}
}

func (s *FakeS3) generateVersionID() string {
	s.nextVersionID++
	return fmt.Sprintf("%04d", s.nextVersionID)
}

// handleRequest handles incoming HTTP requests
func (s *FakeS3) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Parse the bucket and key from the URL path
	// Path format: /{bucket}/{key}
	parts := strings.SplitN(r.URL.Path, "/", 3)
	bucket := ""
	key := ""
	if len(parts) > 1 {
		bucket = parts[1]
	}
	if len(parts) > 2 {
		key = parts[2]
	}

	if s.interceptor != nil {
		if s.interceptor(r, w) {
			return
		}
	}

	if bucket != s.bucket {
		http.Error(w, "Invalid bucket", http.StatusBadRequest)
		return
	}

	// Dispatch based on the HTTP method and query parameters
	switch r.Method {
	case http.MethodHead:
		if key != "" {
			s.handleHeadObject(w, r, key)
		} else {
			http.Error(w, "Not Implmemented", http.StatusNotImplemented)
		}
	case http.MethodGet:
		if key != "" {
			s.handleGetObject(w, r, key)
		} else if _, ok := r.URL.Query()["versions"]; ok {
			s.handleListObjectVersions(w, r, bucket)
		} else {
			http.Error(w, "Not Implmemented", http.StatusNotImplemented)
		}
	case http.MethodPut:
		if _, ok := r.URL.Query()["retention"]; ok {
			s.handlePutObjectRetention(w, r, key)
		} else {
			s.handlePutObject(w, r, key)
		}
	case http.MethodPost:
		if _, ok := r.URL.Query()["delete"]; ok {
			s.handleDeleteObjects(w, r)
		} else {
			http.Error(w, "Not Implemented", http.StatusNotImplemented)
		}
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
