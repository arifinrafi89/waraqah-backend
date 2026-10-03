package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const devJWTSecret = "development-only-secret-do-not-use-in-production-0123456789"

// Load reads the environment (and .env in development) into a Config and validates it.
func Load() (*Config, error) {
	if env := os.Getenv("APP_ENV"); env == "" || env == "development" {
		_ = godotenv.Load() // a missing .env is fine
	}
	return LoadFrom(os.LookupEnv)
}

// LoadFrom is Load with an injectable lookup, for tests.
func LoadFrom(lookup func(string) (string, bool)) (*Config, error) {
	c := &Config{}
	v := reflect.ValueOf(c).Elem()
	t := v.Type()
	var problems []string
	appEnv := "development"
	if s, ok := lookup("APP_ENV"); ok && s != "" {
		appEnv = s
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		key := f.Tag.Get("env")
		raw, ok := lookup(key)
		if !ok || raw == "" {
			raw = f.Tag.Get("default")
		}
		if raw == "" && f.Tag.Get("prod") == "required" && appEnv == "production" {
			problems = append(problems, key+" is required in production")
			continue
		}
		if err := setField(v.Field(i), raw); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", key, err))
		}
	}
	if c.JWTSecret == "" && !c.IsProduction() {
		c.JWTSecret = devJWTSecret
	}
	problems = append(problems, c.validate()...)
	if len(problems) > 0 {
		return nil, errors.New("config: " + strings.Join(problems, "; "))
	}
	return c, nil
}

func (c *Config) validate() []string {
	var p []string
	switch c.AppEnv {
	case "development", "test", "production":
	default:
		p = append(p, "APP_ENV must be development, test or production")
	}
	if len(c.JWTSecret) < 32 {
		p = append(p, "JWT_SECRET must be at least 32 bytes")
	}
	if c.IsProduction() {
		if c.OTPDevCode != "" {
			p = append(p, "OTP_DEV_CODE must be empty in production")
		}
		if c.EmailProvider == "log" {
			p = append(p, "EMAIL_PROVIDER must not be log in production")
		}
		if c.CloudinaryFake {
			p = append(p, "CLOUDINARY_FAKE must be false in production")
		}
	}
	return p
}

func setField(f reflect.Value, raw string) error {
	if f.Kind() == reflect.Slice {
		var out []string
		for _, s := range strings.Split(raw, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		f.Set(reflect.ValueOf(out))
		return nil
	}
	if raw == "" {
		return nil
	}
	switch f.Kind() {
	case reflect.String:
		f.SetString(raw)
	case reflect.Int:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("not an integer: %q", raw)
		}
		f.SetInt(int64(n))
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("not a boolean: %q", raw)
		}
		f.SetBool(b)
	case reflect.Int64: // time.Duration
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("not a duration: %q", raw)
		}
		f.SetInt(int64(d))
	default:
		return fmt.Errorf("unsupported kind %s", f.Kind())
	}
	return nil
}
