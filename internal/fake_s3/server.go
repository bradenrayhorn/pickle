package fakes3

import (
	"fmt"
	"maps"
	"net"
	"net/http"
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
	server        *http.Server
	bucket        string
	objects       map[string]map[string]*ObjectVersion // map[key]map[versionID]*ObjectVersion
	nextVersionID int
	now           time.Time

	boundHost string

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
	s.StartServerWithHostPort("", "")
}

func (s *FakeS3) StartServerWithHostPort(host, port string) {
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "0"
	}
	ln, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		panic("could not bind address" + host + ":" + port)
	}
	s.boundHost = ln.Addr().String()

	s.server = &http.Server{Handler: http.HandlerFunc(s.handleRequest)}

	go func() { _ = s.server.Serve(ln) }()
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
	return s.boundHost
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

func (s *FakeS3) GetByVersionID(versionID string) *ObjectVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, versions := range s.objects {
		for _, version := range versions {
			if version.VersionID == versionID {
				return version
			}
		}
	}

	return nil
}

func (s *FakeS3) GetVersionIDByFuzzyKey(keyContains string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for key, versions := range s.objects {
		if strings.Contains(key, keyContains) {
			if len(versions) == 1 {
				return slices.Collect(maps.Values(versions))[0].VersionID
			} else {
				panic("too many versions at " + key)
			}
		}
	}

	panic("no key containing " + keyContains)
}

func (s *FakeS3) generateVersionID() string {
	s.nextVersionID++
	return fmt.Sprintf("%04d", s.nextVersionID)
}

// handleRequest handles incoming HTTP requests
func (s *FakeS3) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has("probe") {
		_, _ = w.Write([]byte("ok"))
		return
	}

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
