package configresolver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
		"net/url" 

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// keys returns a slice of map keys
func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// newFakeGitLabServer simulates GitLab RepositoryFiles API
func newFakeGitLabServer(t *testing.T, fileBodies map[string][]byte, status map[string]int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the incoming request for debugging
		t.Logf("Request: Path=%s Query=%s", r.URL.Path, r.URL.RawQuery)

		validPrefix := strings.HasPrefix(r.URL.Path, "/projects/") || strings.HasPrefix(r.URL.Path, "/api/v4/projects/")
		if !validPrefix ||
			!strings.Contains(r.URL.Path, "/repository/files/") ||
			!strings.HasSuffix(r.URL.Path, "/raw") {
			http.NotFound(w, r)
			return
		}

		// Support optional /api/v4/ prefix
		parts := strings.Split(r.URL.Path, "/")
		// If path starts with /api/v4/, shift index by 2
		offset := 0
		if len(parts) > 2 && parts[1] == "api" && parts[2] == "v4" {
			offset = 2
		}
		if len(parts) < offset+7 {
			http.NotFound(w, r)
			return
		}
		// Extract project as group/proj
		project := parts[offset+2] + "/" + parts[offset+3]
		// Extract filePath: join all parts between 'repository/files' and 'raw'
		// Find index of 'repository', then 'files', then join everything after 'files' up to 'raw'
		repoIdx := offset+4 // 'repository'
		filesIdx := repoIdx+1 // 'files'
		// filePath is everything from filesIdx+1 up to len(parts)-1 (excluding 'raw')
		filePathParts := parts[filesIdx+1:len(parts)-1]
		filePath := strings.Join(filePathParts, "/")
		project, _ = url.PathUnescape(project)
		filePath, _ = url.PathUnescape(filePath)
		ref := r.URL.Query().Get("ref")

		// Try both encoded and decoded filePath for key lookup
		key := ref + ":" + project + ":" + filePath
		found := false
		if _, ok := fileBodies[key]; ok {
			found = true
		} else {
			// Try with raw (encoded) filePath
			key2 := ref + ":" + project + ":" + strings.Join(filePathParts, "/")
			if _, ok2 := fileBodies[key2]; ok2 {
				key = key2
				found = true
			}
		}
		if !found {
			t.Logf("Mock server: key not found: %q", key)
			available := keys(fileBodies)
			t.Logf("Mock server: available keys: %v", available)
			http.NotFound(w, r)
			return
		}

		if code, ok := status[key]; ok && code != http.StatusOK {
			http.Error(w, "error", code)
			return
		}

		body := fileBodies[key]
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
}


func TestGitLabFetcher_Fetch_Success(t *testing.T) {
	cfg := map[string]interface{}{"rules": map[string]interface{}{"indent": 2.0}}
	b, _ := json.Marshal(cfg)

	fileBodies := map[string][]byte{
		"main:group/proj:.gitlab/renovate.json": b,
	}
	status := map[string]int{}

	srv := newFakeGitLabServer(t, fileBodies, status)
	defer srv.Close()

	client, _ := gitlab.NewClient("dummy-token", gitlab.WithBaseURL(srv.URL), gitlab.WithHTTPClient(srv.Client()))
	f := &GitLabFetcher{Client: client, FilePath: ".gitlab/renovate.json"}

	got, err := f.Fetch("main", "group/proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("Fetch mismatch\nwant: %#v\ngot : %#v", cfg, got)
	}
}

func TestGitLabFetcher_Fetch_NotFound(t *testing.T) {
	fileBodies := map[string][]byte{}
	status := map[string]int{}
	srv := newFakeGitLabServer(t, fileBodies, status)
	defer srv.Close()

	client, _ := gitlab.NewClient("token", gitlab.WithBaseURL(srv.URL), gitlab.WithHTTPClient(srv.Client()))
	f := &GitLabFetcher{Client: client, FilePath: ".gitlab/renovate.json"}

	_, err := f.Fetch("main", "group/proj")
	if err == nil || !strings.Contains(err.Error(), "failed to fetch") {
		t.Fatalf("expected fetch error, got %v", err)
	}
}

func TestGitLabFetcher_Fetch_InvalidJSON(t *testing.T) {
	fileBodies := map[string][]byte{
		"main:group/proj:.gitlab/renovate.json": []byte("{not json"),
	}
	status := map[string]int{
		"main:group/proj:.gitlab/renovate.json": 200, // Ensure 200 OK
	}
	srv := newFakeGitLabServer(t, fileBodies, status)
	defer srv.Close()

	client, _ := gitlab.NewClient("token", gitlab.WithBaseURL(srv.URL), gitlab.WithHTTPClient(srv.Client()))
	f := &GitLabFetcher{Client: client, FilePath: ".gitlab/renovate.json"}

	_, err := f.Fetch("main", "group/proj")
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("expected invalid JSON error, got %v", err)
	}
}

func TestGitLabFetcher_Fetch_HTTPError(t *testing.T) {
	fileBodies := map[string][]byte{}
	status := map[string]int{
		"main:group/proj:.gitlab/renovate.json": 500,
	}
	srv := newFakeGitLabServer(t, fileBodies, status)
	defer srv.Close()

	client, _ := gitlab.NewClient("token", gitlab.WithBaseURL(srv.URL), gitlab.WithHTTPClient(srv.Client()))
	f := &GitLabFetcher{Client: client, FilePath: ".gitlab/renovate.json"}

	_, err := f.Fetch("main", "group/proj")
	if err == nil || !strings.Contains(err.Error(), "failed to fetch") {
		t.Fatalf("expected HTTP error, got %v", err)
	}
}
