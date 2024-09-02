package fs

import (
	"context"
	"fmt"
	"io"
	io_fs "io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aaronland/go-flickr-api/client"
	"github.com/tidwall/gjson"
)

var re_photo = regexp.MustCompile(`^(?:\d+|(?:.*?\#)?\/?\d+\/\d+_\w+_[a-z]\.\w+)$`)
var re_url = regexp.MustCompile(`\#?(\/?\d+\/\d+_\w+_[a-z]\.\w+)$`)

type apiFS struct {
	io_fs.FS
	http_client *http.Client
	client      client.Client
}

// MatchesPhotoId returns a boolean value indicating whether 'v' should be treated as a known Flickr photo ID (or URL)
// or as a "standard photo response" query URL.
func MatchesPhotoId(v string) bool {
	return re_photo.MatchString(v)
}

// MatchesPhotoURL returns boolean value indicating whether 'v' is a valid Flickr photo URL.
func MatchesPhotoURL(v string) bool {
	return re_url.MatchString(v)
}

// DerivePhotoURL extracts a known Flickr photo	URL from a compound path URI as returned by the `fs.ReadDir` method.
func DerivePhotoURL(v string) (string, error) {

	if !MatchesPhotoURL(v) {
		return "", fmt.Errorf("String does not match photo URL")
	}

	m := re_url.FindStringSubmatch(v)
	path := m[1]

	if !strings.HasPrefix(path, "/") {
		path = fmt.Sprintf("/%s", path)
	}

	return path, nil
}

// New creates a new FileSystem that reads files from the Flickr API.
func New(ctx context.Context, cl client.Client) io_fs.FS {

	http_cl := &http.Client{}

	fs := &apiFS{
		http_client: http_cl,
		client:      cl,
	}

	return fs
}

// Open opens the named file. File names are expected to take the form of:
// * A unique numeric identifier for a photo on the Flickr website
// * The fully-qualified path (not the whole URL) for an static photo asset hosted by the Flickr webservers.
// * A URL-encoded query string followed by the fully-qualified path (not the whole URL) for an static	photo asset hosted by the Flickr webservers encoded as a URL fragment.
func (f *apiFS) Open(name string) (io_fs.File, error) {

	ctx := context.Background()

	logger := slog.Default()
	logger = logger.With("name", name)

	logger.Debug("Open file")

	if !MatchesPhotoId(name) {

		logger.Debug("File does not match photo ID or URL, assuming SPR entry")

		fl := &apiFile{
			name:           name,
			content_length: -1,
			modTime:        time.Now(),
			is_spr:         true,
			perm:           0444,
		}

		return fl, nil
	}

	var path string

	if MatchesPhotoURL(name) {

		path, _ = DerivePhotoURL(name)
		logger.Debug("Derive relative path", "rel path", path)
	} else {

		args := &url.Values{}
		args.Set("method", "flickr.photos.getInfo")
		args.Set("photo_id", name)

		logger.Debug("Get photo info", "query", args.Encode())

		r, err := f.client.ExecuteMethod(ctx, args)

		if err != nil {
			return nil, fmt.Errorf("Failed to execute API method, %w", err)
		}

		defer r.Close()

		body, err := io.ReadAll(r)

		if err != nil {
			return nil, fmt.Errorf("Failed to read API response body, %w", err)
		}

		// logger.Debug("API response", "body", string(body))

		id_rsp := gjson.GetBytes(body, "photo.id")

		if !id_rsp.Exists() {
			return nil, fmt.Errorf("Missing photo.id")
		}

		secret_rsp := gjson.GetBytes(body, "photo.secret")

		if !secret_rsp.Exists() {
			return nil, fmt.Errorf("Missing photo.secret")
		}

		originalsecret_rsp := gjson.GetBytes(body, "photo.originalsecret")

		if !originalsecret_rsp.Exists() {
			return nil, fmt.Errorf("Missing photo.originalsecret")
		}

		originalformat_rsp := gjson.GetBytes(body, "photo.originalformat")

		if !originalformat_rsp.Exists() {
			return nil, fmt.Errorf("Missing photo.originalformat")
		}

		server_rsp := gjson.GetBytes(body, "photo.server")

		if !server_rsp.Exists() {
			return nil, fmt.Errorf("Missing photo.server")
		}

		id := id_rsp.Int()
		secret := secret_rsp.String()
		originalsecret := originalsecret_rsp.String()
		originalformat := originalformat_rsp.String()
		server := server_rsp.String()

		if originalsecret != "" {
			path = fmt.Sprintf("/%s/%d_%s_o.%s", server, id, originalsecret, originalformat)
		} else {
			// Something something something small files?
			path = fmt.Sprintf("/%s/%d_%s_b.%s", server, id, secret, "jpg")
		}
	}

	u, err := url.Parse("https://live.staticflickr.com")

	if err != nil {
		return nil, fmt.Errorf("Failed to parse base URL (which is weird), %w", err)
	}

	u.Path = path
	url := u.String()

	logger = logger.With("url", url)
	logger.Debug("Fetch photo")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new request, %w", err)
	}

	rsp, err := f.http_client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("Failed to execute request, %w", err)
	}

	// logger.Debug("Status", "code", rsp.StatusCode)

	if rsp.StatusCode != http.StatusOK {
		defer rsp.Body.Close()
		return nil, fmt.Errorf("%d %s", rsp.StatusCode, rsp.Status)
	}

	str_len := rsp.Header.Get("Content-Length")
	int_len, _ := strconv.ParseInt(str_len, 10, 64)

	// last-modified: Sat, 31 Aug 2024 20:07:47 GMT
	lastmod := rsp.Header.Get("Last-Modified")

	t, err := time.Parse(time.RFC1123, lastmod)

	if err != nil {
		logger.Debug("Failed to parse lastmod time, default to now", "lastmod", lastmod, "error", err)
		t = time.Now()
	}

	// To do: Derive file permissions from Flickr permissions
	// "visibility":{"ispublic":0,"isfriend":0,"isfamily":0}

	fl := &apiFile{
		name:           u.Path,
		content:        rsp.Body,
		content_length: int_len,
		modTime:        t,
	}

	logger.Debug("Return file", "file name", u.Path, "len", int_len)
	return fl, nil
}

// Returns the body of the named file. File names are expected to take the form of:
// * A unique numeric identifier for a photo on the Flickr website
// * The fully-qualified path (not the whole URL) for an static photo asset hosted by the Flickr webservers.
// * A URL-encoded query string followed by the fully-qualified path (not the whole URL) for an static	photo asset hosted by the Flickr webservers encoded as a URL fragment.
func (f apiFS) ReadFile(name string) ([]byte, error) {
	r, err := f.Open(name)

	if err != nil {
		return nil, err
	}

	defer r.Close()
	return io.ReadAll(r)
}

// Return the results of a "standard photo response" Flickr API call as zero or more `fs.DirEntry` instances. File names are
// expected to take the form of a URL-escape URL query string containing the relevant Flickr API call parameters. Fully qualified
// paths (not URLs) for individual photos are returned in query fragments (since the root "directory" is an API call), for example:
//
//	flickr.photosets.getPhotos&photoset_id=1418449&user_id=12037949754%40N01/#/29/65753018_ba12e6eed0_o.jpg
//
// See also: https://code.flickr.net/2008/08/19/standard-photos-response-apis-for-civilized-age/
func (f *apiFS) ReadDir(name string) ([]io_fs.DirEntry, error) {

	logger := slog.Default()
	logger = logger.With("name", name)

	ctx := context.Background()

	args, err := url.ParseQuery(name)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse query, %w", err)
	}

	// https://www.flickr.com/services/api/misc.urls.html

	urls := []string{
		"url_o",  // original
		"url_4k", // has a unique secret; photo owner can restrict (4096)
		"url_f",  // has a unique secret; photo owner can restrict (4096)
		"url_k",  // has a unique secret; photo owner can restrict (2048)
		"url_b",  // 1024
	}

	extras := make([]string, 0)

	ensure_extras := []string{
		"lastupdate",
	}

	for _, u := range urls {
		extras = append(extras, u)
	}

	if args.Has("extras") {
		extras = strings.Split(args.Get("extras"), ",")
		args.Del("extras")
	}

	for _, v := range ensure_extras {

		if !slices.Contains(extras, v) {
			extras = append(extras, v)
		}
	}

	args.Set("extras", strings.Join(extras, ","))

	logger.Debug("Read dir")

	entries := []io_fs.DirEntry{}

	cb := func(ctx context.Context, r io.ReadSeekCloser, err error) error {

		defer r.Close()

		if err != nil {
			return err
		}

		body, err := io.ReadAll(r)

		if err != nil {
			return fmt.Errorf("Failed to read API response body, %w", err)
		}

		// https://code.flickr.net/2008/08/19/standard-photos-response-apis-for-civilized-age/
		rsp := gjson.GetBytes(body, "*.photo")

		if !rsp.Exists() {
			return fmt.Errorf("Failed to derive photos from response")
		}

		for _, ph := range rsp.Array() {

			var ph_url *url.URL

			for _, path := range urls {

				url_rsp := ph.Get(path)

				if !url_rsp.Exists() {
					logger.Warn("Response is missing extra, skipping", "path", path)
					continue
				}

				url_str := url_rsp.String()

				if url_str == "" {
					logger.Warn("Response has empty extra property, skipping", "path", path)
				}

				v, err := url.Parse(url_str)

				if err != nil {
					return fmt.Errorf("Failed to parse url_o value (%s), %w", url_rsp.String(), err)
				}

				ph_url = v
				break
			}

			if ph_url == nil {
				return fmt.Errorf("Failed to derive photo URL")
			}

			lastmod_rsp := ph.Get("lastupdate")
			lastmod := time.Unix(lastmod_rsp.Int(), 0)

			fi := &apiFileInfo{
				name:    fmt.Sprintf("#%s", ph_url.Path),
				size:    -1,
				is_spr:  false,
				modTime: lastmod,
			}

			ent := &apiDirEntry{
				info: fi,
			}

			logger.Debug("Add entry", "path", fi.name)
			entries = append(entries, ent)
		}

		return nil
	}

	err = client.ExecuteMethodPaginatedWithClient(ctx, f.client, &args, cb)

	if err != nil {
		return nil, fmt.Errorf("Failed to execute query, %w", err)
	}

	return entries, nil
}

// Sub returns a "Not supported" error since it is not applicable.
func (f *apiFS) Sub(path string) (io_fs.FS, error) {
	return nil, fmt.Errorf("Not supported")
}
