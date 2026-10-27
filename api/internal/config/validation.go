package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func (config Config) Validate() error {
	return validation.Errors{
		"CORS_ALLOWED_ORIGINS": validation.Validate(config.CORSAllowedOrigins, validation.Required, validation.By(validateCORSOrigins(config.AppEnv))),
		"HOST": validation.Validate(config.Host,
			validation.Required,
		),
		"PORT": validation.Validate(config.Port,
			validation.Required,
			validation.By(validatePort),
		),
		"APP_ENV": validation.Validate(config.AppEnv,
			validation.Required,
			validation.In("development", "production"),
		),
		"FIREBASE_BOOTSTRAP_TIMEOUT_SECONDS": validation.Validate(config.FirebaseBootstrapTimeoutSeconds,
			validation.Required,
			validation.Min(1),
		),
		"FIREBASE_PROJECT_ID": validation.Validate(config.FirebaseProjectID,
			validation.Required,
		),
		"FIREBASE_CREDENTIALS_FILE": validation.Validate(config.FirebaseCredentialsFile,
			validation.By(validateFirebaseCredentials(config.FirebaseCredentialsJSON)),
		),
		"FIREBASE_CREDENTIALS_JSON": validation.Validate(config.FirebaseCredentialsJSON,
			validation.By(validateFirebaseCredentials(config.FirebaseCredentialsFile)),
		),
		"FIREBASE_PING_TIMEOUT_SECONDS": validation.Validate(config.FirebasePingTimeoutSeconds,
			validation.Required,
			validation.Min(1),
		),
		"FIREBASE_STORAGE_BUCKET": validation.Validate(config.FirebaseStorageBucket,
			validation.Required,
		),
		"APP_ROLE": validation.Validate(config.AppRole, validation.Required, validation.In("api", "worker", "all"), validation.By(func(any) error {
			if config.AppRole != "api" && !config.TasksEnabled {
				return fmt.Errorf("worker roles require TASKS_ENABLED=true")
			}
			return nil
		})),
		"TASKS_REDIS_URL": validation.Validate(config.TasksRedisURL, validation.By(func(any) error {
			if config.TasksEnabled && (strings.TrimSpace(config.TasksRedisURL) == "" || sameRedisEndpoint(config.TasksRedisURL, config.RedisURL)) {
				return fmt.Errorf("enabled jobs require a dedicated TASKS_REDIS_URL")
			}
			return nil
		})),
		"TASKS_QUEUE":       validation.Validate(config.TasksQueue, validation.Required),
		"TASKS_CONCURRENCY": validation.Validate(config.TasksConcurrency, validation.Min(1), validation.Max(32)),
		"STUDENT_ABANDONMENT_SECONDS": validation.Validate(config.StudentAbandonmentSeconds,
			validation.Required,
			validation.Min(60),
		),
		"AUTH_GOOGLE_WEB_CLIENT_ID": validation.Validate(config.AuthGoogleWebClientID,
			validation.Required,
		),
		"AUTH_TEACHER_PROVIDERS": validation.Validate(config.AuthTeacherProviders,
			validation.By(validateAuthProviders("google", "password")),
			validation.Required,
		),
		"AUTH_STUDENT_PROVIDERS": validation.Validate(config.AuthStudentProviders,
			validation.By(validateAuthProviders("anonymous")),
			validation.Required,
		),
		"DEMO_TEACHER_EMAIL": validation.Validate(config.DemoTeacherEmail,
			validation.By(validateRequiredWhenEnabled(config.FirebaseBootstrapEnabled)),
		),
		"DEMO_TEACHER_PASSWORD": validation.Validate(config.DemoTeacherPassword,
			validation.By(validateMinimumLengthWhenEnabled(config.FirebaseBootstrapEnabled, 6)),
		),
		"DEMO_TEACHER_DISPLAY_NAME": validation.Validate(config.DemoTeacherDisplayName,
			validation.By(validateRequiredWhenEnabled(config.FirebaseBootstrapEnabled)),
		),
		"GENAI_BACKEND": validation.Validate(config.GenAIBackend,
			validation.By(validateGenAIBackend(config.GenAIEnabled)),
		),
		"GENAI_MODEL": validation.Validate(config.GenAIModel,
			validation.By(validateRequiredWhenEnabled(config.GenAIEnabled)),
		),
		"GENAI_API_KEY": validation.Validate(config.GenAIAPIKey,
			validation.By(validateGenAIAPIKey(config.GenAIEnabled, config.GenAIBackend)),
		),
		"GENAI_PROJECT_ID": validation.Validate(config.GenAIProjectID,
			validation.By(validateGenAIProject(config.GenAIEnabled, config.GenAIBackend)),
		),
		"GENAI_LOCATION": validation.Validate(config.GenAILocation,
			validation.By(validateGenAIProject(config.GenAIEnabled, config.GenAIBackend)),
		),
		"GENAI_CREDENTIALS_FILE": validation.Validate(config.GenAICredentialsFile,
			validation.By(validateGenAICredentials(config.GenAICredentialsJSON)),
		),
		"GENAI_CREDENTIALS_JSON": validation.Validate(config.GenAICredentialsJSON,
			validation.By(validateGenAICredentials(config.GenAICredentialsFile)),
		),
		"GENAI_REQUEST_TIMEOUT_SECONDS": validation.Validate(config.GenAIRequestTimeoutSeconds,
			validation.Required,
			validation.Min(1),
		),
		"GENAI_MAX_OUTPUT_TOKENS": validation.Validate(config.GenAIMaxOutputTokens,
			validation.Required,
			validation.Min(256),
		),
		"POSTHOG_PROJECT_TOKEN": validation.Validate(config.PostHogProjectToken,
			validation.By(validateRequiredWhenEnabled(config.PostHogEnabled)),
		),
		"POSTHOG_HOST": validation.Validate(config.PostHogHost,
			validation.By(validateHTTPSURL(config.PostHogEnabled, "PostHog")),
		),
		"REDIS_URL": validation.Validate(config.RedisURL,
			validation.Required,
			validation.By(validateRedisURL),
		),
		"REDIS_CA_CERT_PATH": validation.Validate(config.RedisCACertPath,
			validation.By(validateRedisCACert(config.RedisCACertPEM)),
		),
		"REDIS_CA_CERT_PEM": validation.Validate(config.RedisCACertPEM,
			validation.By(validateRedisCACert(config.RedisCACertPath)),
		),
		"REDIS_PING_TIMEOUT_SECONDS": validation.Validate(config.RedisPingTimeoutSeconds,
			validation.Required,
			validation.Min(1),
		),
	}.Filter()
}

func validatePort(value any) error {
	port, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	parsedPort, err := strconv.Atoi(port)
	if err != nil || parsedPort < 1 || parsedPort > 65535 {
		return fmt.Errorf("must be a valid TCP port between 1 and 65535")
	}

	return nil
}

func validateFirebaseCredentials(otherValue string) validation.RuleFunc {
	return func(value any) error {
		credentialsValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}

		credentialsValue = strings.TrimSpace(credentialsValue)
		otherValue = strings.TrimSpace(otherValue)
		if credentialsValue == "" && otherValue == "" {
			return fmt.Errorf("must configure FIREBASE_CREDENTIALS_FILE or FIREBASE_CREDENTIALS_JSON")
		}
		if credentialsValue != "" && otherValue != "" {
			return fmt.Errorf("must configure either FIREBASE_CREDENTIALS_FILE or FIREBASE_CREDENTIALS_JSON, not both")
		}

		return nil
	}
}

func validateGenAIBackend(enabled bool) validation.RuleFunc {
	return func(value any) error {
		if !enabled {
			return nil
		}

		backend, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if backend != "vertex" && backend != "gemini_api" {
			return fmt.Errorf("must be vertex or gemini_api")
		}

		return nil
	}
}

func validateRequiredWhenEnabled(enabled bool) validation.RuleFunc {
	return func(value any) error {
		if !enabled {
			return nil
		}

		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("is required when its feature is enabled")
		}

		return nil
	}
}

func validateMinimumLengthWhenEnabled(enabled bool, minimum int) validation.RuleFunc {
	return func(value any) error {
		if !enabled {
			return nil
		}

		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if len(text) < minimum {
			return fmt.Errorf("must contain at least %d characters when its feature is enabled", minimum)
		}

		return nil
	}
}

func validateHTTPSURL(enabled bool, featureName string) validation.RuleFunc {
	return func(value any) error {
		if !enabled {
			return nil
		}

		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		parsed, err := url.Parse(strings.TrimSpace(text))
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("must be an absolute https URL when %s is enabled", featureName)
		}

		return nil
	}
}

func validateGenAIAPIKey(enabled bool, backend string) validation.RuleFunc {
	return func(value any) error {
		if !enabled || backend != "gemini_api" {
			return nil
		}

		apiKey, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if strings.TrimSpace(apiKey) == "" {
			return fmt.Errorf("is required when GENAI_BACKEND=gemini_api")
		}

		return nil
	}
}

func validateGenAIProject(enabled bool, backend string) validation.RuleFunc {
	return func(value any) error {
		if !enabled || backend != "vertex" {
			return nil
		}

		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("is required when GENAI_BACKEND=vertex")
		}

		return nil
	}
}

func validateGenAICredentials(otherValue string) validation.RuleFunc {
	return func(value any) error {
		credentialsValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if strings.TrimSpace(credentialsValue) != "" && strings.TrimSpace(otherValue) != "" {
			return fmt.Errorf("must configure either GENAI_CREDENTIALS_FILE or GENAI_CREDENTIALS_JSON, not both")
		}

		return nil
	}
}

func validateRedisURL(value any) error {
	redisURL, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if strings.TrimSpace(redisURL) == "" {
		return nil
	}
	if hasPlaceholder(redisURL) {
		return fmt.Errorf("must replace placeholders with the real Redis connection string")
	}
	if !strings.HasPrefix(redisURL, "redis://") && !strings.HasPrefix(redisURL, "rediss://") {
		return fmt.Errorf("must start with redis:// or rediss://")
	}

	return nil
}

func validateRedisCACert(otherValue string) validation.RuleFunc {
	return func(value any) error {
		certValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}

		certValue = strings.TrimSpace(certValue)
		if certValue == "" {
			return nil
		}
		if strings.TrimSpace(otherValue) != "" {
			return fmt.Errorf("must configure either REDIS_CA_CERT_PATH or REDIS_CA_CERT_PEM, not both")
		}

		return nil
	}
}

func hasPlaceholder(value string) bool {
	return strings.Contains(value, "<") || strings.Contains(value, ">")
}

func validateAuthProviders(allowed ...string) validation.RuleFunc {
	return func(value any) error {
		configured, _ := value.(string)
		for _, provider := range strings.Split(configured, ",") {
			supported := false
			for _, candidate := range allowed {
				if strings.TrimSpace(provider) == candidate {
					supported = true
				}
			}
			if !supported {
				return fmt.Errorf("unsupported authentication provider %q", strings.TrimSpace(provider))
			}
		}
		return nil
	}
}
func validateCORSOrigins(environment string) validation.RuleFunc {
	return func(value any) error {
		origins, _ := value.(string)
		for _, origin := range strings.Split(origins, ",") {
			parsed, err := url.Parse(strings.TrimSpace(origin))
			if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" || strings.Contains(origin, "*") {
				return fmt.Errorf("CORS origins must be exact HTTP(S) origins")
			}
			if parsed.Scheme != "https" && (environment == "production" || parsed.Scheme != "http") {
				return fmt.Errorf("production CORS origins must use HTTPS")
			}
		}
		return nil
	}
}

// Different database numbers do not turn a disposable cache into a dedicated broker.
func sameRedisEndpoint(first, second string) bool {
	left, err := url.Parse(strings.TrimSpace(first))
	if err != nil || left.Host == "" {
		return false
	}
	right, err := url.Parse(strings.TrimSpace(second))
	if err != nil || right.Host == "" {
		return false
	}
	port := func(value *url.URL) string {
		if value.Port() != "" {
			return value.Port()
		}
		return "6379"
	}
	return strings.EqualFold(left.Hostname(), right.Hostname()) && port(left) == port(right)
}
