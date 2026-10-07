package images

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/tomasweigenast/pdfgen/internal/testutil"
	"github.com/tomasweigenast/pdfgen/internal/urls"
)

type fakeResource struct {
	mime string
	body []byte
	err  error
}

type fakeFetcher struct {
	resources map[string]fakeResource
	fetched   []string
}

func (f *fakeFetcher) Fetch(_ context.Context, url string, _ http.Header) (*urls.Response, error) {
	f.fetched = append(f.fetched, url)
	res, ok := f.resources[url]
	if !ok {
		return nil, &urls.FetchError{URL: url, Err: errors.New("not found")}
	}
	if res.err != nil {
		return nil, res.err
	}
	return &urls.Response{URL: url, Status: 200, MIMEType: res.mime, Body: io.NopCloser(bytes.NewReader(res.body))}, nil
}

func newTestGetter(t *testing.T, resources map[string]fakeResource) (Getter, *fakeFetcher) {
	t.Helper()
	f := &fakeFetcher{resources: resources}
	return NewGetter(context.Background(), f, nil, Options{}), f
}

func TestGetterCachesByURLAndOrientation(t *testing.T) {
	get, fetcher := newTestGetter(t, map[string]fakeResource{
		"a.png": {"image/png", readResource(t, "pattern.png"), nil},
		"b.png": {"image/png", readResource(t, "pattern.png"), nil},
	})
	first, err := get("a.png", "", OrientationFromImage)
	if err != nil || first == nil {
		t.Fatalf("get = %v, %v", first, err)
	}
	if again, _ := get("a.png", "", OrientationFromImage); again != first {
		t.Error("the same URL and orientation gave another image")
	}
	rotated, _ := get("a.png", "", Orientation{Angle: 90})
	if rotated == first {
		t.Error("another orientation gave the cached image")
	}
	other, _ := get("b.png", "", OrientationFromImage)
	if other == first {
		t.Error("another URL gave the cached image")
	}
	if got := len(fetcher.fetched); got != 3 {
		t.Errorf("%d fetches, want 3", got)
	}

	ids := map[string]bool{}
	for _, img := range []Image{first, rotated, other} {
		ids[img.(*Raster).ID()] = true
	}
	if len(ids) != 3 {
		t.Errorf("images share XObject names: %v", ids)
	}
	sum := md5.Sum([]byte("a.png"))
	if got, want := first.(*Raster).ID(), hex.EncodeToString(sum[:]); got != want {
		t.Errorf("ID = %s, want %s", got, want)
	}
}

func TestGetterWarnsOnceAboutBrokenImages(t *testing.T) {
	get, fetcher := newTestGetter(t, map[string]fakeResource{
		"garbage.png": {"image/png", []byte("this is not an image"), nil},
		"vector.svg":  {"image/svg+xml", []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"), nil},
	})
	logs := testutil.CaptureLogs(t)
	for _, url := range []string{"missing.png", "garbage.png", "vector.svg"} {
		for i := 0; i < 2; i++ {
			img, err := get(url, "", OrientationFromImage)
			if img != nil || err != nil {
				t.Fatalf("get(%s) = %v, %v, want nil, nil", url, img, err)
			}
		}
	}
	if got := len(logs.Warnings()); got != 3 {
		t.Errorf("%d warnings, want one per URL: %q", got, logs.Warnings())
	}
	if got := len(fetcher.fetched); got != 3 {
		t.Errorf("%d fetches, want 3", got)
	}
}

func TestGetterIgnoresTheAnnouncedMediaType(t *testing.T) {
	png := readResource(t, "pattern.png")
	get, _ := newTestGetter(t, map[string]fakeResource{
		"liar.jpg":    {"image/jpeg", png, nil},
		"unknown":     {"text/plain", png, nil},
		"forced.html": {"text/html", png, nil},
	})
	for _, url := range []string{"liar.jpg", "unknown", "forced.html"} {
		if img, _ := get(url, "image/png", OrientationFromImage); img == nil {
			t.Errorf("%s was not decoded", url)
		}
	}
}

func TestGetterSVGMediaTypeFallsBackToBitmap(t *testing.T) {
	get, _ := newTestGetter(t, map[string]fakeResource{
		"really-a-png.svg": {"image/svg+xml", readResource(t, "pattern.png"), nil},
	})
	if img, _ := get("really-a-png.svg", "", OrientationFromImage); img == nil {
		t.Error("a PNG announced as SVG was not decoded")
	}
}

func TestGetterStopsOnFatalErrors(t *testing.T) {
	fatal := &urls.FatalError{URL: "x.png", Err: errors.New("down")}
	get, _ := newTestGetter(t, map[string]fakeResource{"x.png": {err: fatal}})
	img, err := get("x.png", "", OrientationFromImage)
	if img != nil || !urls.IsFatal(err) {
		t.Errorf("get = %v, %v, want a fatal error", img, err)
	}
}

func TestGetterSharesACacheBetweenGetters(t *testing.T) {
	f := &fakeFetcher{resources: map[string]fakeResource{"a.png": {"image/png", readResource(t, "pattern.png"), nil}}}
	cache := NewCache()
	first, _ := NewGetter(context.Background(), f, cache, Options{})("a.png", "", OrientationFromImage)
	second, _ := NewGetter(context.Background(), f, cache, Options{})("a.png", "", OrientationFromImage)
	if first == nil || first != second || len(f.fetched) != 1 {
		t.Errorf("fetched %d times", len(f.fetched))
	}
}

func TestGetterReadsFilesAndDataURLs(t *testing.T) {
	get := NewGetter(context.Background(), urls.NewDefaultFetcher(), nil, Options{})
	file, err := get(testutil.ResourceURL(t, "pattern.png"), "", OrientationFromImage)
	if err != nil || file == nil {
		t.Fatalf("file URL: %v, %v", file, err)
	}
	w, h, _ := file.IntrinsicSize(1, 16)
	if w.V != 4 || h.V != 4 {
		t.Errorf("size %v×%v", w.V, h.V)
	}
	inline, err := get("data:image/png;base64,"+base64.StdEncoding.EncodeToString(readResource(t, "pattern.png")), "", OrientationFromImage)
	if err != nil || inline == nil {
		t.Fatalf("data URL: %v, %v", inline, err)
	}
}
