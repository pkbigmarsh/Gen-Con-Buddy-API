package bgg

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"
)

const defaultBGGBaseURL = "https://boardgamegeek.com"

const defaultBGGTimeout = 60 * time.Second

// Credentials are the BGG account login passed to Fetcher.FetchRanksCSV to
// authenticate before downloading the ranks data dump.
type Credentials struct {
	Username string
	Password string
}

// Fetcher downloads the BGG "Board Game Ranks" data dump. The zero value is not
// usable; construct it with NewFetcher (or set HTTPClient and BaseURL directly
// in tests).
type Fetcher struct {
	HTTPClient *http.Client // cookiejar-backed session client
	BaseURL    string       // BGG site root, e.g. https://boardgamegeek.com
}

// NewFetcher returns a Fetcher with a cookiejar-backed client and the default
// BGG base URL.
func NewFetcher() (*Fetcher, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	return &Fetcher{
		HTTPClient: &http.Client{Jar: jar, Timeout: defaultBGGTimeout},
		BaseURL:    defaultBGGBaseURL,
	}, nil
}

// loginPayload is the JSON body for BGG's login API.
type loginPayload struct {
	Credentials loginCredentials `json:"credentials"`
}

type loginCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ranksURLPattern matches the presigned ranks-dump link in the data-dumps page.
// Anchored on the distinctive path so it ignores any other links and works with
// a test server host.
var ranksURLPattern = regexp.MustCompile(`https?://[^"'\s]+/boardgames_export/boardgames_ranks_\d{4}-\d{2}-\d{2}\.zip[^"'\s]*`)

// FetchRanksCSV logs in, locates the presigned ranks-dump URL, downloads and
// unzips it, and returns the CSV as a ReadCloser. The caller must Close it.
func (f *Fetcher) FetchRanksCSV(ctx context.Context, creds Credentials) (io.ReadCloser, error) {
	if creds.Username == "" || creds.Password == "" {
		return nil, fmt.Errorf("bgg credentials are required (username and password)")
	}

	if err := f.login(ctx, creds); err != nil {
		return nil, err
	}

	dumpURL, err := f.findRanksURL(ctx)
	if err != nil {
		return nil, err
	}

	return f.downloadCSV(ctx, dumpURL)
}

func (f *Fetcher) login(ctx context.Context, creds Credentials) error {
	payload := loginPayload{
		Credentials: loginCredentials{Username: creds.Username, Password: creds.Password},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal login payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.BaseURL+"/login/api/v1", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("bgg login request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bgg login failed: unexpected status %d", resp.StatusCode)
	}

	if f.HTTPClient.Jar != nil {
		f.HTTPClient.Jar.SetCookies(req.URL, keepCookies(resp.Cookies()))
	}

	return nil
}

// keepCookies filters out deletion cookies (Max-Age <= 0, surfaced as a negative
// MaxAge by net/http) and empty-valued cookies. BGG's login response sets the
// auth cookies both as real values and as Max-Age=0 deletions; a cookiejar would
// otherwise let the deletion clobber the real cookie.
func keepCookies(cs []*http.Cookie) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(cs))
	for _, c := range cs {
		if c.MaxAge < 0 || c.Value == "" {
			continue
		}

		out = append(out, c)
	}

	return out
}

func (f *Fetcher) findRanksURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.BaseURL+"/data_dumps/bg_ranks", nil)
	if err != nil {
		return "", fmt.Errorf("build data-dumps request: %w", err)
	}

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch data-dumps page: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("data-dumps page returned status %d", resp.StatusCode)
	}

	page, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read data-dumps page: %w", err)
	}

	match := ranksURLPattern.FindString(string(page))
	if match == "" {
		return "", fmt.Errorf("ranks dump link not found on data-dumps page (login may have failed)")
	}

	return html.UnescapeString(match), nil
}

func (f *Fetcher) downloadCSV(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build dump download request: %w", err)
	}

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download ranks dump: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ranks dump download returned status %d", resp.StatusCode)
	}

	zipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ranks dump zip: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open ranks dump zip: %w", err)
	}

	for _, file := range zr.File {
		if strings.HasSuffix(file.Name, ".csv") {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("open csv entry %q: %w", file.Name, err)
			}

			return rc, nil
		}
	}

	return nil, fmt.Errorf("no csv entry found in ranks dump zip")
}
