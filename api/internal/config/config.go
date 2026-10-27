package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"classgame-api/internal/demoaccount"
)

type Config struct {
	TrustedProxyCIDRs               string
	AuthTeacherEmailAllowlist       string
	CORSAllowedOrigins              string
	Host                            string
	Port                            string
	AppEnv                          string
	FirebaseBootstrapEnabled        bool
	FirebaseBootstrapTimeoutSeconds int
	FirebaseProjectID               string
	FirebaseCredentialsFile         string
	FirebaseCredentialsJSON         string
	FirebasePingTimeoutSeconds      int
	FirebaseStorageBucket           string
	AppRole                         string
	TasksEnabled                    bool
	TasksRedisURL                   string
	TasksQueue                      string
	TasksConcurrency                int
	StudentAbandonmentSeconds       int
	AuthTeacherProviders            string
	AuthStudentProviders            string
	DemoTeacherEmail                string
	DemoTeacherPassword             string
	DemoTeacherDisplayName          string
	GenAIEnabled                    bool
	GenAIBackend                    string
	GenAIModel                      string
	GenAIAPIKey                     string
	GenAIProjectID                  string
	GenAILocation                   string
	GenAICredentialsFile            string
	GenAICredentialsJSON            string
	GenAIRequestTimeoutSeconds      int
	GenAIMaxOutputTokens            int
	GenAISharedBudgetScope          string
	GenAISharedConcurrent           int
	GenAISharedRequestsPerMinute    int
	GenAISharedOutputPerMinute      int
	GenAISharedLeaseSeconds         int
	PostHogEnabled                  bool
	PostHogProjectToken             string
	PostHogHost                     string
	RedisURL                        string
	RedisCACertPath                 string
	RedisCACertPEM                  string
	RedisPingTimeoutSeconds         int
}

func Load() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	reader := envReader{}
	googleApplicationCredentials := reader.getString("GOOGLE_APPLICATION_CREDENTIALS", "")
	googleCloudProject := reader.getString("GOOGLE_CLOUD_PROJECT", "rare-tape-466414-f4")
	googleCloudLocation := reader.getString("GOOGLE_CLOUD_LOCATION", "global")
	firebaseProjectID := reader.getString("FIREBASE_PROJECT_ID", "allthingsagentichackatho-ee359")
	corsDefault := "http://localhost:3000,http://127.0.0.1:3000"
	if reader.getString("APP_ENV", "development") == "production" {
		corsDefault = ""
	}
	config := Config{
		TrustedProxyCIDRs:               reader.getString("TRUSTED_PROXY_CIDRS", ""),
		AuthTeacherEmailAllowlist:       reader.getString("AUTH_TEACHER_EMAIL_ALLOWLIST", ""),
		CORSAllowedOrigins:              reader.getString("CORS_ALLOWED_ORIGINS", corsDefault),
		Host:                            reader.getString("HOST", "0.0.0.0"),
		Port:                            reader.getString("PORT", "4004"),
		AppEnv:                          reader.getString("APP_ENV", "development"),
		FirebaseBootstrapEnabled:        reader.getBool("FIREBASE_BOOTSTRAP_ENABLED", false),
		FirebaseBootstrapTimeoutSeconds: reader.getInt("FIREBASE_BOOTSTRAP_TIMEOUT_SECONDS", 30),
		FirebaseProjectID:               firebaseProjectID,
		FirebaseCredentialsFile:         reader.getString("FIREBASE_CREDENTIALS_FILE", ""),
		FirebaseCredentialsJSON:         reader.getString("FIREBASE_CREDENTIALS_JSON", ""),
		FirebasePingTimeoutSeconds:      reader.getInt("FIREBASE_PING_TIMEOUT_SECONDS", 5),
		FirebaseStorageBucket:           reader.getString("FIREBASE_STORAGE_BUCKET", "allthingsagentichackatho-ee359.firebasestorage.app"),
		AppRole:                         reader.getString("APP_ROLE", "api"),
		TasksEnabled:                    reader.getBool("TASKS_ENABLED", false),
		TasksRedisURL:                   reader.getString("TASKS_REDIS_URL", ""),
		TasksQueue:                      reader.getString("TASKS_QUEUE", "classgame-background"),
		TasksConcurrency:                reader.getInt("TASKS_CONCURRENCY", 4),
		StudentAbandonmentSeconds:       reader.getInt("STUDENT_ABANDONMENT_SECONDS", 120),
		AuthTeacherProviders:            reader.getString("AUTH_TEACHER_PROVIDERS", "google"),
		AuthStudentProviders:            reader.getString("AUTH_STUDENT_PROVIDERS", "anonymous"),
		DemoTeacherEmail:                reader.getString("DEMO_TEACHER_EMAIL", demoaccount.DefaultEmail),
		DemoTeacherPassword:             reader.getString("DEMO_TEACHER_PASSWORD", ""),
		DemoTeacherDisplayName:          reader.getString("DEMO_TEACHER_DISPLAY_NAME", demoaccount.DefaultDisplayName),
		GenAIEnabled:                    reader.getBool("GENAI_ENABLED", true),
		GenAIBackend:                    reader.getString("GENAI_BACKEND", "vertex"),
		GenAIModel:                      reader.getString("GENAI_MODEL", "gemini-3.7-flash"),
		GenAIAPIKey:                     reader.getString("GENAI_API_KEY", reader.getString("GEMINI_API_KEY", reader.getString("GOOGLE_API_KEY", ""))),
		GenAIProjectID:                  reader.getString("GENAI_PROJECT_ID", googleCloudProject),
		GenAILocation:                   reader.getString("GENAI_LOCATION", googleCloudLocation),
		GenAICredentialsFile:            reader.getString("GENAI_CREDENTIALS_FILE", googleApplicationCredentials),
		GenAICredentialsJSON:            reader.getString("GENAI_CREDENTIALS_JSON", ""),
		GenAIRequestTimeoutSeconds:      reader.getInt("GENAI_REQUEST_TIMEOUT_SECONDS", 30),
		GenAIMaxOutputTokens:            reader.getInt("GENAI_MAX_OUTPUT_TOKENS", 4096),
		GenAISharedBudgetScope:          reader.getString("GENAI_SHARED_BUDGET_SCOPE", "classgame-provider"),
		GenAISharedConcurrent:           reader.getInt("GENAI_SHARED_CONCURRENT", 4),
		GenAISharedRequestsPerMinute:    reader.getInt("GENAI_SHARED_REQUESTS_PER_MINUTE", 60),
		GenAISharedOutputPerMinute:      reader.getInt("GENAI_SHARED_OUTPUT_TOKENS_PER_MINUTE", 65536),
		GenAISharedLeaseSeconds:         reader.getInt("GENAI_SHARED_LEASE_SECONDS", 150),
		PostHogEnabled:                  reader.getBool("POSTHOG_ENABLED", false),
		PostHogProjectToken:             reader.getString("POSTHOG_PROJECT_TOKEN", ""),
		PostHogHost:                     reader.getString("POSTHOG_HOST", "https://us.i.posthog.com"),
		RedisURL:                        reader.getString("REDIS_URL", ""),
		RedisCACertPath:                 reader.getString("REDIS_CA_CERT_PATH", ""),
		RedisCACertPEM:                  reader.getString("REDIS_CA_CERT_PEM", ""),
		RedisPingTimeoutSeconds:         reader.getInt("REDIS_PING_TIMEOUT_SECONDS", 5),
	}

	if err := reader.validate(); err != nil {
		return Config{}, err
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (config Config) Address() string {
	return fmt.Sprintf("%s:%s", config.Host, config.Port)
}

func loadDotEnv() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}
