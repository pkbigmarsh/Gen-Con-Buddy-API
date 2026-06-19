package bgg

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFetcher_Defaults(t *testing.T) {
	f, err := NewFetcher()
	require.NoError(t, err)
	require.NotNil(t, f.HTTPClient)
	require.NotNil(t, f.HTTPClient.Jar)
	require.Equal(t, defaultBGGBaseURL, f.BaseURL)
}

// buildRanksZip returns an in-memory zip containing a boardgames_ranks.csv with
// the given CSV body.
func buildRanksZip(t *testing.T, csvBody string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("boardgames_ranks.csv")
	require.NoError(t, err)
	_, err = w.Write([]byte(csvBody))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestFetchRanksCSV_HappyPath(t *testing.T) {
	const csvBody = "id,name,yearpublished,rank,bayesaverage,average,usersrated,is_expansion\n1,Wingspan,2019,10,8.0,8.2,50000,0\n"
	zipBytes := buildRanksZip(t, csvBody)

	mux := http.NewServeMux()
	mux.HandleFunc("/login/api/v1", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SessionID", Value: "test-session"})
		w.WriteHeader(http.StatusNoContent)
	})
	var dumpHref string
	mux.HandleFunc("/data_dumps/bg_ranks", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`<html><body><a href="` + dumpHref + `?X-Amz-Signature=abc&amp;X-Amz-Expires=900">Download</a></body></html>`)); err != nil {
			t.Errorf("write page: %v", err)
		}
	})
	mux.HandleFunc("/boardgames_export/boardgames_ranks_2026-05-29.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		if _, err := w.Write(zipBytes); err != nil {
			t.Errorf("write zip: %v", err)
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	dumpHref = srv.URL + "/boardgames_export/boardgames_ranks_2026-05-29.zip"

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	f := &Fetcher{HTTPClient: &http.Client{Jar: jar}, BaseURL: srv.URL}

	rc, err := f.FetchRanksCSV(t.Context(), Credentials{Username: "u", Password: "p"})
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, csvBody, string(got))
}

// TestFetchRanksCSV_KeepsAuthCookies reproduces BGG's login behaviour where an
// auth cookie is set once as a real value and again as a Max-Age=0 deletion.
// A naive cookiejar drops the real cookie; the fetcher must retain it so the
// authenticated data-dumps page (which requires the cookie) returns the link.
func TestFetchRanksCSV_KeepsAuthCookies(t *testing.T) {
	const csvBody = "id,name\n1,Wingspan\n"
	zipBytes := buildRanksZip(t, csvBody)

	mux := http.NewServeMux()
	mux.HandleFunc("/login/api/v1", func(w http.ResponseWriter, r *http.Request) {
		// real auth cookie ...
		http.SetCookie(w, &http.Cookie{Name: "bggusername", Value: "u", Path: "/", MaxAge: 2592000})
		// ... immediately followed by a deletion duplicate (same name/path) that
		// clobbers it in a naive jar.
		w.Header().Add("Set-Cookie", "bggusername=; Path=/; Max-Age=0")
		http.SetCookie(w, &http.Cookie{Name: "SessionID", Value: "s", Path: "/"})
		w.WriteHeader(http.StatusNoContent)
	})
	var dumpHref string
	mux.HandleFunc("/data_dumps/bg_ranks", func(w http.ResponseWriter, r *http.Request) {
		// Only an authenticated request (auth cookie present) gets the link.
		if ck, err := r.Cookie("bggusername"); err != nil || ck.Value != "u" {
			_, _ = w.Write([]byte(`<html>not logged in</html>`))
			return
		}
		_, _ = w.Write([]byte(`<html><a href="` + dumpHref + `">d</a></html>`))
	})
	mux.HandleFunc("/boardgames_export/boardgames_ranks_2026-05-29.zip", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBytes)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	dumpHref = srv.URL + "/boardgames_export/boardgames_ranks_2026-05-29.zip"

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	f := &Fetcher{HTTPClient: &http.Client{Jar: jar}, BaseURL: srv.URL}

	rc, err := f.FetchRanksCSV(t.Context(), Credentials{Username: "u", Password: "p"})
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, csvBody, string(got))
}

func TestFetchRanksCSV_Errors(t *testing.T) {
	emptyZip := func(t *testing.T) []byte {
		t.Helper()
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("readme.txt")
		require.NoError(t, err)
		_, err = w.Write([]byte("not a csv"))
		require.NoError(t, err)
		require.NoError(t, zw.Close())
		return buf.Bytes()
	}

	tests := []struct {
		name      string
		creds     Credentials
		loginCode int    // status returned by /login/api/v1
		pageBody  string // %s is replaced with the dump URL
		dumpCode  int    // status returned by the zip endpoint
		dumpZip   func(t *testing.T) []byte
		wantErr   string
	}{
		{
			name:    "missing credentials",
			creds:   Credentials{Username: "", Password: ""},
			wantErr: "credentials are required",
		},
		{
			name:      "login rejected",
			creds:     Credentials{Username: "u", Password: "p"},
			loginCode: http.StatusForbidden,
			wantErr:   "login failed",
		},
		{
			name:      "link missing",
			creds:     Credentials{Username: "u", Password: "p"},
			loginCode: http.StatusNoContent,
			pageBody:  `<html><body>no link here</body></html>`,
			wantErr:   "ranks dump link not found",
		},
		{
			name:      "dump download fails",
			creds:     Credentials{Username: "u", Password: "p"},
			loginCode: http.StatusNoContent,
			pageBody:  `<html><a href="%s">d</a></html>`,
			dumpCode:  http.StatusForbidden,
			wantErr:   "download returned status 403",
		},
		{
			name:      "zip has no csv",
			creds:     Credentials{Username: "u", Password: "p"},
			loginCode: http.StatusNoContent,
			pageBody:  `<html><a href="%s">d</a></html>`,
			dumpCode:  http.StatusOK,
			dumpZip:   emptyZip,
			wantErr:   "no csv entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/login/api/v1", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.loginCode)
			})
			var dumpURL string
			mux.HandleFunc("/data_dumps/bg_ranks", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(strings.Replace(tt.pageBody, "%s", dumpURL, 1)))
			})
			mux.HandleFunc("/boardgames_export/boardgames_ranks_2026-05-29.zip", func(w http.ResponseWriter, r *http.Request) {
				if tt.dumpCode != http.StatusOK {
					w.WriteHeader(tt.dumpCode)
					return
				}
				if tt.dumpZip != nil {
					_, _ = w.Write(tt.dumpZip(t))
				}
			})

			srv := httptest.NewServer(mux)
			defer srv.Close()
			dumpURL = srv.URL + "/boardgames_export/boardgames_ranks_2026-05-29.zip"

			jar, err := cookiejar.New(nil)
			require.NoError(t, err)
			f := &Fetcher{HTTPClient: &http.Client{Jar: jar}, BaseURL: srv.URL}

			_, err = f.FetchRanksCSV(t.Context(), tt.creds)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
