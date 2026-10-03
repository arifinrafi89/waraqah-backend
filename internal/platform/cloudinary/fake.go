package cloudinary

import (
	"context"
	"strings"
	"sync"
)

// Fake stores nothing and answers example.invalid URLs (tests and development without keys).
type Fake struct {
	Folder string

	mu        sync.Mutex
	uploaded  map[string]bool
	destroyed []string
}

// NewFake builds the fake uploader.
func NewFake(folder string) *Fake { return &Fake{Folder: folder, uploaded: map[string]bool{}} }

// Upload pretends to store the image.
func (f *Fake) Upload(_ context.Context, kind, id, slot string, _ Image) (Result, error) {
	publicID := strings.Join([]string{f.Folder, kind, id, slot}, "/")
	f.mu.Lock()
	f.uploaded[publicID] = true
	f.mu.Unlock()
	return Result{URL: "https://example.invalid/" + publicID, PublicID: publicID}, nil
}

// Destroy records the removal.
func (f *Fake) Destroy(_ context.Context, publicID string) error {
	f.mu.Lock()
	delete(f.uploaded, publicID)
	f.destroyed = append(f.destroyed, publicID)
	f.mu.Unlock()
	return nil
}

// Has reports whether a public id is currently "stored".
func (f *Fake) Has(publicID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.uploaded[publicID]
}

// Destroyed lists the public ids removed so far.
func (f *Fake) Destroyed() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.destroyed...)
}
