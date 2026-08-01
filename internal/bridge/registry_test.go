package bridge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDispatchPrefixMatching(t *testing.T) {
	r := NewRegistry()
	r.Register("GET", "/api/health", func(w http.ResponseWriter, req *http.Request) { w.Write([]byte("health")) })
	r.Register("GET", "/api/conversations/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("conv:" + strings.TrimPrefix(req.URL.Path, "/api/conversations/")))
	})
	r.Register("GET", "/api/fs/list", func(w http.ResponseWriter, req *http.Request) { w.Write([]byte("fslist")) })

	cases := []struct {
		method, path, want string
		ok                 bool
	}{
		{"GET", "/api/health", "health", true},
		{"GET", "/api/fs/list?path=C:\\x", "fslist", true},
		{"GET", "/api/conversations/conv_1/messages", "conv:conv_1/messages", true},
		{"GET", "/api/conversations/conv_1", "conv:conv_1", true},
		{"GET", "/api/nonexistent", "", false},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(c.method, c.path, nil)
		ok := r.Dispatch(c.method, strings.Split(c.path, "?")[0], rec, req)
		if ok != c.ok {
			t.Errorf("%s %s: ok=%v want %v", c.method, c.path, ok, c.ok)
			continue
		}
		if ok && rec.Body.String() != c.want {
			t.Errorf("%s %s: body=%q want %q", c.method, c.path, rec.Body.String(), c.want)
		}
	}
}
