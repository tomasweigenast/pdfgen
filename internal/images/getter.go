package images

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/tomasweigenast/pdfgen/internal/logging"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

// Getter loads the image at url, or returns nil when it cannot be loaded; the
// failure has been logged and the document is rendered without the image.
// forcedMIME, when set, replaces the media type the server announced. The
// error is non-nil only when fetching failures must stop the rendering.
type Getter func(url, forcedMIME string, orientation Orientation) (Image, error)

type cacheKey struct {
	url         string
	orientation Orientation
}

// Cache remembers loaded images, and the failures to load them, by URL and
// orientation. The zero value is not usable; call NewCache.
type Cache struct {
	mu     sync.Mutex
	images map[cacheKey]Image
}

// NewCache returns an empty cache.
func NewCache() *Cache { return &Cache{images: make(map[cacheKey]Image)} }

func (c *Cache) get(k cacheKey) (Image, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	img, ok := c.images[k]
	return img, ok
}

func (c *Cache) put(k cacheKey, img Image) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.images[k] = img
}

// NewGetter returns a Getter that fetches with fetcher and stores in cache,
// which is created when nil. opts shape how raster images are stored.
func NewGetter(ctx context.Context, fetcher urls.Fetcher, cache *Cache, opts Options) Getter {
	if cache == nil {
		cache = NewCache()
	}
	return func(url, forcedMIME string, orientation Orientation) (Image, error) {
		key := cacheKey{url, orientation}
		if img, ok := cache.get(key); ok {
			return img, nil
		}
		img, err := load(ctx, fetcher, url, forcedMIME, orientation, opts)
		if err != nil {
			if urls.IsFatal(err) {
				return nil, err
			}
			logging.Warnf("Failed to load image at %q: %s", url, err)
			img = nil
		}
		cache.put(key, img)
		return img, nil
	}
}

func load(ctx context.Context, fetcher urls.Fetcher, url, forcedMIME string, orientation Orientation, opts Options) (Image, error) {
	resp, err := fetcher.Fetch(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	mime := forcedMIME
	if mime == "" {
		mime = resp.MIMEType
	}

	// A file announced as SVG is tried as SVG first; anything else is tried
	// as a bitmap first and as SVG only if that fails.
	var svgErr error
	if mime == "image/svg+xml" {
		var img Image
		if img, svgErr = decodeSVG(data, url); svgErr == nil {
			return img, nil
		}
	}
	raster, err := NewRaster(imageID(url, orientation), data, orientation, opts)
	if err == nil {
		return raster, nil
	}
	if mime == "image/svg+xml" {
		return nil, svgErr
	}
	if img, svgErr := decodeSVG(data, url); svgErr == nil {
		return img, nil
	}
	return nil, err
}

// decodeSVG is the entry point for vector images, which are not supported yet.
var decodeSVG = func(data []byte, url string) (Image, error) {
	return nil, errors.New("SVG images are not supported")
}

// imageID names an image after its URL, and after its orientation when that
// differs from the file's own, so that each variant has its own XObject.
func imageID(url string, orientation Orientation) string {
	key := url
	if orientation != OrientationFromImage {
		key = fmt.Sprintf("%s\x00%s", url, orientation)
	}
	sum := md5.Sum([]byte(key))
	return hex.EncodeToString(sum[:])
}
