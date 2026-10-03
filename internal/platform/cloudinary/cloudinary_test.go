package cloudinary

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	pngBytes = append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 32)...)
	b64      = func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
)

func TestDecodeImageAcceptsRealImages(t *testing.T) {
	img, err := DecodeImage(b64(pngBytes), 1<<20)
	if err != nil || img.ContentType != "image/png" {
		t.Fatalf("%v %v", img.ContentType, err)
	}
	if _, err := DecodeImage("data:image/png;base64,"+b64(pngBytes), 1<<20); err != nil {
		t.Errorf("data URI refused: %v", err)
	}
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 32)...)
	if img, err := DecodeImage(b64(jpeg), 1<<20); err != nil || img.ContentType != "image/jpeg" {
		t.Errorf("jpeg: %v %v", img.ContentType, err)
	}
}

func TestDecodeImageRejects(t *testing.T) {
	notImage := []byte("<html><script>alert(1)</script></html>")
	cases := []struct {
		name string
		in   string
		max  int
		want error
	}{
		{"renamed non-image", b64(notImage), 1 << 20, ErrNotAnImage},
		{"too big", b64(pngBytes), 10, ErrImageTooBig},
		{"huge text", strings.Repeat("A", 100000), 100, ErrImageTooBig},
		{"not base64", "%%%not base64%%%", 1 << 20, ErrNotBase64},
		{"empty", "", 1 << 20, ErrEmptyImage},
	}
	for _, c := range cases {
		if _, err := DecodeImage(c.in, c.max); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v want %v", c.name, err, c.want)
		}
	}
}

func TestThumb(t *testing.T) {
	in := "https://res.cloudinary.com/demo/image/upload/v1/waraqah/p2p/x/front.jpg"
	want := "https://res.cloudinary.com/demo/image/upload/c_fill,w_300,q_auto,f_auto/v1/waraqah/p2p/x/front.jpg"
	if got := Thumb(in); got != want {
		t.Errorf("got %s", got)
	}
	if got := Thumb("https://example.invalid/a.jpg"); got != "https://example.invalid/a.jpg" {
		t.Errorf("non-cloudinary url changed: %s", got)
	}
}

func TestRealSignsAndUploads(t *testing.T) {
	var gotForm map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = map[string]string{}
		for k := range r.PostForm {
			gotForm[k] = r.PostForm.Get(k)
		}
		if strings.HasSuffix(r.URL.Path, "/destroy") {
			_, _ = w.Write([]byte(`{"result":"ok"}`))
			return
		}
		_, _ = w.Write([]byte(`{"secure_url":"https://res.cloudinary.com/demo/image/upload/v1/` + gotForm["public_id"] + `"}`))
	}))
	defer srv.Close()
	r := NewReal("demo", "key123", "secret456", "waraqah-dev")
	r.BaseURL = srv.URL
	r.Now = func() time.Time { return time.Unix(1700000000, 0) }

	res, err := r.Upload(context.Background(), "p2p", "p2p-1", "front", Image{Bytes: pngBytes, ContentType: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	if res.PublicID != "waraqah-dev/p2p/p2p-1/front" || !strings.Contains(res.URL, res.PublicID) {
		t.Errorf("result %+v", res)
	}
	// The signature covers the signed params, sorted, plus the secret; never the file or api_key.
	want := r.sign(map[string]string{"public_id": res.PublicID, "overwrite": "true", "invalidate": "true", "timestamp": "1700000000"})
	if gotForm["signature"] != want || gotForm["api_key"] != "key123" || strings.Contains(gotForm["signature"], "secret456") {
		t.Errorf("form: %v", gotForm)
	}
	if err := r.Destroy(context.Background(), res.PublicID); err != nil {
		t.Errorf("destroy: %v", err)
	}
}

func TestRealUploadFailureIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	r := NewReal("demo", "k", "s", "f")
	r.BaseURL = srv.URL
	if _, err := r.Upload(context.Background(), "p2p", "x", "front", Image{Bytes: pngBytes, ContentType: "image/png"}); !errors.Is(err, ErrUploadFailed) {
		t.Errorf("got %v", err)
	}
}

func TestFake(t *testing.T) {
	f := NewFake("waraqah-dev")
	res, _ := f.Upload(context.Background(), "p2p", "p2p-1", "back", Image{})
	if res.URL != "https://example.invalid/waraqah-dev/p2p/p2p-1/back" || !f.Has(res.PublicID) {
		t.Errorf("%+v", res)
	}
	_ = f.Destroy(context.Background(), res.PublicID)
	if f.Has(res.PublicID) || len(f.Destroyed()) != 1 {
		t.Error("destroy not recorded")
	}
}
