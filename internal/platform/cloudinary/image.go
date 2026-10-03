// Package cloudinary stores photos. The app sends them as base64 inside JSON; the server checks
// them and uploads them with a signed request (BACKEND_PLAN.md section 11).
package cloudinary

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

// Errors a handler turns into a refusal.
var (
	ErrNotBase64    = errors.New("image is not valid base64")
	ErrNotAnImage   = errors.New("image is not a JPEG, PNG or WebP")
	ErrImageTooBig  = errors.New("image is too large")
	ErrEmptyImage   = errors.New("image is empty")
	ErrUploadFailed = errors.New("image upload failed")
)

// Image is a checked, decoded photo.
type Image struct {
	Bytes       []byte
	ContentType string
}

// Result is where an uploaded photo lives.
type Result struct {
	URL      string
	PublicID string
}

// Uploader stores and removes photos. Real and Fake implement it.
type Uploader interface {
	// Upload stores img as <folder>/<kind>/<id>/<slot> and returns its URL and public id.
	Upload(ctx context.Context, kind, id, slot string, img Image) (Result, error)
	// Destroy removes an uploaded photo (best effort for callers).
	Destroy(ctx context.Context, publicID string) error
}

// DecodeImage decodes base64 (a data: URI prefix is allowed), enforces maxBytes, and checks the
// bytes are a JPEG, PNG or WebP. The file name or claimed type is never trusted.
func DecodeImage(b64 string, maxBytes int) (Image, error) {
	if i := strings.Index(b64, ","); strings.HasPrefix(b64, "data:") && i >= 0 {
		b64 = b64[i+1:]
	}
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return Image{}, ErrEmptyImage
	}
	// Reject before decoding when the encoded text alone is already over the limit.
	if base64.StdEncoding.DecodedLen(len(b64)) > maxBytes+3 {
		return Image{}, ErrImageTooBig
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		if raw, err = base64.RawStdEncoding.DecodeString(b64); err != nil {
			return Image{}, ErrNotBase64
		}
	}
	if len(raw) == 0 {
		return Image{}, ErrEmptyImage
	}
	if len(raw) > maxBytes {
		return Image{}, ErrImageTooBig
	}
	ct := http.DetectContentType(raw)
	switch ct {
	case "image/jpeg", "image/png", "image/webp":
		return Image{Bytes: raw, ContentType: ct}, nil
	}
	return Image{}, ErrNotAnImage
}

// Thumb turns an uploaded image URL into a small version using a URL transformation.
// URLs that are not Cloudinary delivery URLs are returned unchanged.
func Thumb(url string) string {
	const marker = "/upload/"
	i := strings.Index(url, marker)
	if i < 0 || !strings.Contains(url, "cloudinary.com") {
		return url
	}
	return url[:i+len(marker)] + "c_fill,w_300,q_auto,f_auto/" + url[i+len(marker):]
}
