package app

import authfeature "github.com/arifinrafi89/waraqah-backend/internal/feature/auth"

// Each feature gets its service here, with the platform services and cross-feature interfaces it needs.

func authService(d *Deps) *authfeature.Service {
	return &authfeature.Service{
		DB: d.DB, JWT: d.JWT, Refresh: d.Refresh, OTP: d.OTP, Google: d.Google,
		Email: d.Email, Clock: d.Clock, Loc: d.Location, Cost: d.Cfg.BcryptCost, Log: d.Log,
	}
}
