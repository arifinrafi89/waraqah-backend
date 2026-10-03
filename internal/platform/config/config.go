// Package config loads every setting from environment variables (BACKEND_PLAN.md §18).
// Each field has an `env` tag with its variable name; .env.example must list the same names.
package config

import "time"

// Config is the whole backend configuration.
type Config struct {
	// App
	AppEnv             string   `env:"APP_ENV" default:"development"`
	Port               string   `env:"PORT" default:"8080"`
	APIBasePath        string   `env:"API_BASE_PATH" default:"/v1"`
	PublicBaseURL      string   `env:"PUBLIC_BASE_URL" default:"http://localhost:8080"`
	AppTimezone        string   `env:"APP_TIMEZONE" default:"Asia/Dhaka"`
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" prod:"required"`
	MaxRequestBodyMB   int      `env:"MAX_REQUEST_BODY_MB" default:"12"`
	MaxImageMB         int      `env:"MAX_IMAGE_MB" default:"4"`

	// Logging
	LogLevel  string `env:"LOG_LEVEL" default:"info"`
	LogFormat string `env:"LOG_FORMAT" default:"text"`
	DebugSQL  bool   `env:"DEBUG_SQL" default:"false"`

	// Database
	DatabaseURL          string `env:"DATABASE_URL" prod:"required"`
	DatabaseURLDirect    string `env:"DATABASE_URL_DIRECT"`
	DatabaseURLTest      string `env:"DATABASE_URL_TEST"`
	DBMaxConns           int    `env:"DB_MAX_CONNS" default:"5"`
	RunMigrationsOnStart bool   `env:"RUN_MIGRATIONS_ON_START" default:"false"`

	// Auth
	JWTSecret            string        `env:"JWT_SECRET" prod:"required"`
	JWTIssuer            string        `env:"JWT_ISSUER" default:"waraqah"`
	JWTAccessTTL         time.Duration `env:"JWT_ACCESS_TTL" default:"15m"`
	JWTRefreshTTL        time.Duration `env:"JWT_REFRESH_TTL" default:"720h"`
	BcryptCost           int           `env:"BCRYPT_COST" default:"12"`
	OTPTTL               time.Duration `env:"OTP_TTL" default:"10m"`
	OTPMaxAttempts       int           `env:"OTP_MAX_ATTEMPTS" default:"5"`
	OTPResendSeconds     int           `env:"OTP_RESEND_SECONDS" default:"60"`
	OTPDevCode           string        `env:"OTP_DEV_CODE"`
	GoogleOAuthClientIDs []string      `env:"GOOGLE_OAUTH_CLIENT_IDS" prod:"required"`

	// Email
	EmailProvider string `env:"EMAIL_PROVIDER" default:"log"`
	EmailAPIKey   string `env:"EMAIL_API_KEY" prod:"required"`
	EmailFrom     string `env:"EMAIL_FROM" prod:"required"`

	// Images (Cloudinary)
	CloudinaryCloudName string `env:"CLOUDINARY_CLOUD_NAME" prod:"required"`
	CloudinaryAPIKey    string `env:"CLOUDINARY_API_KEY" prod:"required"`
	CloudinaryAPISecret string `env:"CLOUDINARY_API_SECRET" prod:"required"`
	CloudinaryFolder    string `env:"CLOUDINARY_FOLDER" default:"waraqah-dev"`
	CloudinaryFake      bool   `env:"CLOUDINARY_FAKE" default:"true"`

	// AI assistant
	GeminiAPIKey  string        `env:"GEMINI_API_KEY"`
	GeminiModel   string        `env:"GEMINI_MODEL"`
	GeminiTimeout time.Duration `env:"GEMINI_TIMEOUT" default:"6s"`
	AIRatePerMin  int           `env:"AI_RATE_PER_MIN" default:"10"`

	// Rate limits
	AuthRatePerMin    int `env:"AUTH_RATE_PER_MIN" default:"20"`
	ReportRatePerHour int `env:"REPORT_RATE_PER_HOUR" default:"30"`

	// Demo behaviour and jobs
	DemoMode           bool          `env:"DEMO_MODE" default:"true"`
	DemoBotDelay       time.Duration `env:"DEMO_BOT_DELAY" default:"4s"`
	CourierPickupDelay time.Duration `env:"COURIER_PICKUP_DELAY" default:"10m"`
	JobsTick           time.Duration `env:"JOBS_TICK" default:"1m"`

	// Seeding
	SeedDemoPassword    string `env:"SEED_DEMO_PASSWORD"`
	SeedAllowProduction bool   `env:"SEED_ALLOW_PRODUCTION" default:"false"`

	// Local tools
	FrontendDir string `env:"FRONTEND_DIR" default:"../waraqah-frontend"`
	APIBaseURL  string `env:"API_BASE_URL" default:"http://localhost:8080/v1"`
}

// IsProduction reports whether the server runs in production.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// IsDevelopment reports whether the server runs in development.
func (c *Config) IsDevelopment() bool { return c.AppEnv == "development" }
