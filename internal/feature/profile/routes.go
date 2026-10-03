package profile

import (
	"net/http"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/httpx"
)

// Routes registers the profile endpoints. Each line names the fake API constant it serves.
func (h *Handler) Routes(r httpx.Router, a *auth.Middleware) {
	r.Handle("GET /profile", a.Me(http.HandlerFunc(h.Profile)))                   // ProfileFakeApi.profile
	r.Handle("POST /profile/save", a.Me(http.HandlerFunc(h.SaveProfile)))         // ProfileFakeApi.saveProfile
	r.Handle("GET /profile/prefs", a.Me(http.HandlerFunc(h.Prefs)))               // ProfileFakeApi.prefs
	r.Handle("POST /profile/prefs/save", a.Me(http.HandlerFunc(h.SavePrefs)))     // ProfileFakeApi.savePrefs
	r.Handle("GET /addresses", a.Me(http.HandlerFunc(h.Addresses)))               // ProfileFakeApi.addresses
	r.Handle("POST /addresses/save", a.Me(http.HandlerFunc(h.SaveAddress)))       // ProfileFakeApi.saveAddress
	r.Handle("POST /addresses/default", a.Me(http.HandlerFunc(h.DefaultAddress))) // ProfileFakeApi.defaultAddress
	r.Handle("POST /addresses/delete", a.Me(http.HandlerFunc(h.DeleteAddress)))   // ProfileFakeApi.deleteAddress
	r.Handle("GET /geo", a.Public(http.HandlerFunc(h.Geo)))                       // ProfileFakeApi.geo
	r.Handle("POST /auth/delete", a.Me(http.HandlerFunc(h.DeleteAccount)))        // ProfileFakeApi.deleteAccount
}
