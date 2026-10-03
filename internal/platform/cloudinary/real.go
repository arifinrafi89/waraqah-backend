package cloudinary

import (
	"context"
	"crypto/sha1" //nolint:gosec // Cloudinary's signature scheme is SHA-1
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Real uploads with Cloudinary's signed Admin/Upload API.
type Real struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
	BaseURL   string // default https://api.cloudinary.com/v1_1 (tests point it at a local server)
	Client    *http.Client
	Now       func() time.Time
}

// NewReal builds the real uploader.
func NewReal(cloudName, apiKey, apiSecret, folder string) *Real {
	return &Real{CloudName: cloudName, APIKey: apiKey, APISecret: apiSecret, Folder: folder,
		BaseURL: "https://api.cloudinary.com/v1_1", Client: &http.Client{Timeout: 30 * time.Second}, Now: time.Now}
}

// sign makes the Cloudinary signature: sorted key=value pairs joined by &, then the secret, SHA-1.
func (r *Real) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "&") + r.APISecret)) //nolint:gosec
	return hex.EncodeToString(sum[:])
}

func (r *Real) post(ctx context.Context, action string, signed map[string]string, extra url.Values) (map[string]any, error) {
	signed["timestamp"] = strconv.FormatInt(r.Now().Unix(), 10)
	form := url.Values{}
	for k, v := range signed {
		form.Set(k, v)
	}
	for k, v := range extra {
		form[k] = v
	}
	form.Set("api_key", r.APIKey)
	form.Set("signature", r.sign(signed))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/image/%s", r.BaseURL, r.CloudName, action), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUploadFailed, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUploadFailed, resp.StatusCode)
	}
	return out, nil
}

// Upload stores the image under <folder>/<kind>/<id>/<slot>, replacing an older one.
func (r *Real) Upload(ctx context.Context, kind, id, slot string, img Image) (Result, error) {
	publicID := strings.Join([]string{r.Folder, kind, id, slot}, "/")
	out, err := r.post(ctx, "upload",
		map[string]string{"public_id": publicID, "overwrite": "true", "invalidate": "true"},
		url.Values{"file": {"data:" + img.ContentType + ";base64," + base64.StdEncoding.EncodeToString(img.Bytes)}})
	if err != nil {
		return Result{}, err
	}
	u, _ := out["secure_url"].(string)
	if u == "" {
		return Result{}, ErrUploadFailed
	}
	return Result{URL: u, PublicID: publicID}, nil
}

// Destroy deletes an uploaded photo.
func (r *Real) Destroy(ctx context.Context, publicID string) error {
	_, err := r.post(ctx, "destroy", map[string]string{"public_id": publicID, "invalidate": "true"}, nil)
	return err
}
