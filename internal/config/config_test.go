package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_WithDefaults(t *testing.T) {
	// Clear all environment variables first
	clearTestEnv()

	// Set only required variables
	os.Setenv("SONARR_URL", "http://test-sonarr:8989")
	os.Setenv("SONARR_API_KEY", "test-api-key")
	defer clearTestEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Test required fields
	if config.Sonarr.URL != "http://test-sonarr:8989" {
		t.Errorf("Expected Sonarr URL 'http://test-sonarr:8989', got '%s'", config.Sonarr.URL)
	}
	if config.Sonarr.APIKey != "test-api-key" {
		t.Errorf("Expected Sonarr API key 'test-api-key', got '%s'", config.Sonarr.APIKey)
	}

	// Test defaults
	if config.RequestTimeout != 30*time.Second {
		t.Errorf("Expected RequestTimeout '30s', got '%v'", config.RequestTimeout)
	}
	if config.RequestDelay != 500*time.Millisecond {
		t.Errorf("Expected RequestDelay '500ms', got '%v'", config.RequestDelay)
	}
	if config.ConcurrentLimit != 5 {
		t.Errorf("Expected ConcurrentLimit '5', got '%d'", config.ConcurrentLimit)
	}
	if config.LogLevel != "INFO" {
		t.Errorf("Expected LogLevel 'INFO', got '%s'", config.LogLevel)
	}
	if config.DryRun {
		t.Errorf("Expected DryRun 'false', got '%t'", config.DryRun)
	}
}

func TestLoadConfig_WithCustomValues(t *testing.T) {
	clearTestEnv()

	// Set all environment variables
	os.Setenv("SONARR_URL", "https://custom-sonarr.com:9999")
	os.Setenv("SONARR_API_KEY", "custom-key-123")
	os.Setenv("REQUEST_TIMEOUT", "60s")
	os.Setenv("REQUEST_DELAY", "1s")
	os.Setenv("CONCURRENT_LIMIT", "10")
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("DRY_RUN", "true")
	defer clearTestEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Test all custom values
	if config.Sonarr.URL != "https://custom-sonarr.com:9999" {
		t.Errorf("Expected Sonarr URL 'https://custom-sonarr.com:9999', got '%s'", config.Sonarr.URL)
	}
	if config.Sonarr.APIKey != "custom-key-123" {
		t.Errorf("Expected Sonarr API key 'custom-key-123', got '%s'", config.Sonarr.APIKey)
	}
	if config.RequestTimeout != 60*time.Second {
		t.Errorf("Expected RequestTimeout '60s', got '%v'", config.RequestTimeout)
	}
	if config.RequestDelay != time.Second {
		t.Errorf("Expected RequestDelay '1s', got '%v'", config.RequestDelay)
	}
	if config.ConcurrentLimit != 10 {
		t.Errorf("Expected ConcurrentLimit '10', got '%d'", config.ConcurrentLimit)
	}
	if config.LogLevel != "DEBUG" {
		t.Errorf("Expected LogLevel 'DEBUG', got '%s'", config.LogLevel)
	}
	if !config.DryRun {
		t.Errorf("Expected DryRun 'true', got '%t'", config.DryRun)
	}
}

func TestLoadConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "missing SONARR_API_KEY only",
			envVars: map[string]string{"SONARR_URL": "http://test:8989"},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "configuration validation failed: SONARR_API_KEY is required"
			},
		},
		{
			name: "empty SONARR_API_KEY", 
			envVars: map[string]string{
				"SONARR_URL":     "http://test:8989",
				"SONARR_API_KEY": "",
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "configuration validation failed: SONARR_API_KEY is required"
			},
		},
		{
			name: "no env vars set",
			envVars: map[string]string{},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "configuration validation failed: SONARR_API_KEY is required"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnv()
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearTestEnv()

			_, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("LoadConfig() error = %v, did not match expected pattern", err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Sonarr: SonarrConfig{
					URL:    "http://test:8989",
					APIKey: "test-key",
				},
				RequestTimeout:  30 * time.Second,
				ConcurrentLimit: 5,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: &Config{
				Sonarr: SonarrConfig{
					APIKey: "test-key",
				},
				RequestTimeout:  30 * time.Second,
				ConcurrentLimit: 5,
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			config: &Config{
				Sonarr: SonarrConfig{
					URL: "http://test:8989",
				},
				RequestTimeout:  30 * time.Second,
				ConcurrentLimit: 5,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: &Config{
				Sonarr: SonarrConfig{
					URL:    "http://test:8989",
					APIKey: "test-key",
				},
				RequestTimeout:  0,
				ConcurrentLimit: 5,
			},
			wantErr: true,
		},
		{
			name: "zero concurrent limit",
			config: &Config{
				Sonarr: SonarrConfig{
					URL:    "http://test:8989",
					APIKey: "test-key",
				},
				RequestTimeout:  30 * time.Second,
				ConcurrentLimit: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "env var set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			setEnv:       true,
			expected:     "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_VAR_MISSING",
			defaultValue: "default",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "env var empty string",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		setEnv       bool
		expected     bool
	}{
		{
			name:         "env var true",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "env var false",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			setEnv:       true,
			expected:     false,
		},
		{
			name:         "env var not set",
			key:          "TEST_BOOL_MISSING",
			defaultValue: true,
			setEnv:       false,
			expected:     true,
		},
		{
			name:         "env var invalid",
			key:          "TEST_BOOL_INVALID",
			defaultValue: false,
			envValue:     "not-a-bool",
			setEnv:       true,
			expected:     false, // should return default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvBool() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// clearTestEnv clears all environment variables that might affect tests
func clearTestEnv() {
	envVars := []string{
		"SONARR_URL", "SONARR_API_KEY",
		"RADARR_URL", "RADARR_API_KEY",
		"REQUEST_TIMEOUT", "REQUEST_DELAY", "CONCURRENT_LIMIT",
		"LOG_LEVEL", "DRY_RUN",
	}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
