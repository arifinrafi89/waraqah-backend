package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestDefaultsAndParsing(t *testing.T) {
	c, err := LoadFrom(env(map[string]string{
		"JWT_ACCESS_TTL":       "20m",
		"CORS_ALLOWED_ORIGINS": "http://a.test, http://b.test",
		"DEMO_MODE":            "false",
		"DB_MAX_CONNS":         "9",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.JWTAccessTTL != 20*time.Minute || c.DBMaxConns != 9 || c.DemoMode || c.Port != "8080" {
		t.Errorf("unexpected config: %+v", c)
	}
	if len(c.CORSAllowedOrigins) != 2 || c.CORSAllowedOrigins[1] != "http://b.test" {
		t.Errorf("origins: %v", c.CORSAllowedOrigins)
	}
}

func TestProductionRules(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"APP_ENV": "production", "OTP_DEV_CODE": "123456", "JWT_SECRET": "short"}))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"DATABASE_URL is required", "OTP_DEV_CODE must be empty", "JWT_SECRET must be at least 32", "EMAIL_PROVIDER must not be log"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}

func TestBadValueNamesVariable(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"JWT_ACCESS_TTL": "soon"}))
	if err == nil || !strings.Contains(err.Error(), "JWT_ACCESS_TTL") {
		t.Fatalf("got %v", err)
	}
}
