# Experimental io/fs.FS support

There is now experimental support for reading photos and the results of [standard places response](https://code.flickr.net/2008/08/19/standard-photos-response-apis-for-civilized-age/) API calls using the [io/fs.FS](https://pkg.go.dev/io/fs) interfaces.

Individual photos can be read using the `Open` method where file names are expected to take the form of.

* A unique numeric identifier for a photo on the Flickr website
* The fully-qualified path (not the whole URL) for an static photo asset hosted by the Flickr webservers.
* A URL-encoded query string followed by the fully-qualified path (not the whole URL) for an static	photo asset hosted by the Flickr webservers encoded as a URL fragment.

For example:

```
fs.Open("6923069836")
```

Or:

```
fs.Open("/7129/7070018487_fbd223e965_o.png")
```

Or:

```
fs.Open("method=flickr.photosets.getPhotos&photoset_id=72157629455113026&user_id=35034348999%40N01/#/7244/7071114647_b8bcd16b65_o.jpg")
```

The last form is to account for paths derived from the `fs.ReadDir` method. Specifically, the `ReadDir` method expects filenames to take the form of URL-escaped URL query parameters to be passed to the Flickr API. For example:

```
fs.ReadDir("method=flickr.photosets.getPhotos&photoset_id=72157629455113026&user_id=35034348999%40N01")
```

This method will return zero or more `fs.DirEntry` instances whose name (path) will be "#" + the fully qualified URL to the photo matching the query. When combined with the directory "root" which is actually a set of query parameters you end up with things like:

```
method=flickr.photosets.getPhotos&photoset_id=72157629455113026&user_id=35034348999%40N01/#/7244/7071114647_b8bcd16b65_o.jpg
```

Which is not ideal but easy enough to account for (which the `Open` and `ReadFile` methods do automatically.

## Tests

All of the [tests](fs_test.go) pass but there may still be "gotchas" or other edge cases. In order to run the tests with calls to the Flickr API you will need to run them with a valid `-client-uri` flag. For example:

```
$> go test -v -run TestFS -client-uri 'oauth1://?consumer_key={CONSUMER_KEY}&consumer_secret={CONSUMER_SECRET}&oauth_token={OAUTH_TOKEN}&oauth_token_secret={OAUTH_SECRET}'
```

## Caching

There is current no caching. Every time you `Open` a file it is fetched from the Flickr API and/or photo servers. Some amount of caching would be good.

## Example

_Error handling has been removed for the sake of brevity._

```
import (
	"context"
	"image"	
	io_fs "io/fs"
	
	"github.com/aaronland/go-flickr-api/client"
	"github.com/aaronland/go-flickr-api/fs"
)

func main() {

	ctx := context.Background()

	// https://github.com/aaronland/go-flickr-api?tab=readme-ov-file#clients
	client_uri := "oauth1://..."
	
	cl, _ := client.NewClient(ctx, client_uri)
	fs := New(ctx, cl)

	r, _ := fs.Open("6923069836")
	defer r.Close()

	im, _, _ := image.Decode(fl)
	// Do something with im here

	walk_func := func(path string, d io_fs.DirEntry, err error) error {

		r, _ := fs.Open(path)
		defer r.Close()

		// Do something with r here
		return nil
	}

	u := url.Values{}
	u.Set("method", "flickr.photosets.getPhotos")
	u.Set("photoset_id", "72157629455113026")
	u.Set("user_id", "35034348999@N01")
	
	q := u.Encode()

	io_fs.WalkDir(fs, q, walk_func)
)	
```