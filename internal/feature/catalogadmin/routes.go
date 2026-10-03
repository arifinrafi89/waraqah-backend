package catalogadmin

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Routes registers the 25 Admin → Catalog endpoints. Each line names the fake API constant it
// serves; every one needs the staff catalog permission (a reader gets 403, a guest 401).
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	staff := func(f http.HandlerFunc) http.Handler { return a.StaffCan(auth.PermCatalog, f) }

	r.Handle("POST /admin/catalog/books/save", staff(h.SaveBook)) // CatalogAdminFakeApi.saveBook
	r.Handle("POST /admin/catalog/books/hide", staff(h.HideBook)) // CatalogAdminFakeApi.hideBook
	for _, kind := range []struct{ name, kind string }{{"categories", "category"}, {"authors", "author"}, {"publishers", "publisher"}} {
		base := "/admin/catalog/" + kind.name
		r.Handle("GET "+base, staff(h.Records(kind.kind)))                 // CatalogAdminFakeApi.records(kind)
		r.Handle("POST "+base+"/save", staff(h.SaveRecord(kind.kind)))     // CatalogAdminFakeApi.records(kind) + /save
		r.Handle("POST "+base+"/delete", staff(h.DeleteRecord(kind.kind))) // CatalogAdminFakeApi.records(kind) + /delete
	}
	r.Handle("GET /admin/catalog/banners", staff(h.Banners))                      // CatalogAdminFakeApi.banners
	r.Handle("POST /admin/catalog/banners/save", staff(h.SaveBanner))             // CatalogAdminFakeApi.saveBanner
	r.Handle("POST /admin/catalog/banners/delete", staff(h.DeleteBanner))         // CatalogAdminFakeApi.deleteBanner
	r.Handle("POST /admin/catalog/banners/move", staff(h.MoveBanner))             // CatalogAdminFakeApi.moveBanner
	r.Handle("GET /admin/catalog/season", staff(h.Season))                        // CatalogAdminFakeApi.season
	r.Handle("POST /admin/catalog/season/save", staff(h.SaveSeason))              // CatalogAdminFakeApi.season + /save
	r.Handle("POST /admin/catalog/collections/save", staff(h.SaveCollection))     // CatalogAdminFakeApi.saveCollection
	r.Handle("POST /admin/catalog/collections/delete", staff(h.DeleteCollection)) // CatalogAdminFakeApi.deleteCollection
	r.Handle("POST /admin/catalog/booklists/save", staff(h.SaveBooklist))         // CatalogAdminFakeApi.saveBooklist
	r.Handle("POST /admin/catalog/booklists/delete", staff(h.DeleteBooklist))     // CatalogAdminFakeApi.deleteBooklist
	r.Handle("GET /admin/catalog/isbn-lookup", staff(h.IsbnLookup))               // CatalogToolsFakeApi.isbnLookup
	r.Handle("GET /admin/catalog/low-stock", staff(h.LowStock))                   // CatalogToolsFakeApi.lowStock
	r.Handle("POST /admin/catalog/editions/stock", staff(h.EditionStock))         // CatalogToolsFakeApi.editionStock
	r.Handle("POST /admin/catalog/import", staff(h.Import))                       // CatalogToolsFakeApi.importBooks
}
