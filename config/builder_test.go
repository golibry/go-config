package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestConfigSuite is the test suite for config functionality
type ConfigTestSuite struct {
	suite.Suite
}

// DatabaseConfig represents a database configuration that implements Config
type DatabaseConfig struct {
	Host     string `validate:"required"`
	Port     int    `validate:"min=1,max=65535"`
	Username string `validate:"required"`
	Password string `validate:"required"`
}

// Populate implements the Config interface for DatabaseConfig
func (d *DatabaseConfig) Populate() error {
	d.Host = os.Getenv("DB_HOST")
	d.Username = os.Getenv("DB_USERNAME")
	d.Password = os.Getenv("DB_PASSWORD")
	d.Port, _ = strconv.Atoi(os.Getenv("DB_PORT"))

	return nil
}

// RedisConfig represents a Redis configuration that implements Config
type RedisConfig struct {
	Host string `validate:"required"`
	Port int    `validate:"min=1,max=65535"`
}

// Populate implements the Config interface for RedisConfig
func (r *RedisConfig) Populate() error {
	r.Host = os.Getenv("REDIS_HOST")
	r.Port, _ = strconv.Atoi(os.Getenv("REDIS_PORT"))

	return nil
}

// AppConfig is a composite configuration struct
type AppConfig struct {
	Database DatabaseConfig `validate:"required"`
	Redis    RedisConfig    `validate:"required"`
	AppName  string         `validate:"required"`
}

// FailingConfig is a config that fails during populating
type FailingConfig struct {
	Value string
}

func (f *FailingConfig) Populate() error {
	return errors.New("populate failed")
}

// CompositeWithFailingConfig contains a config that fails
type CompositeWithFailingConfig struct {
	Failing FailingConfig
}

// TestItCanPopulateAndValidateCompositeConfig tests the main functionality
func (suite *ConfigTestSuite) TestItCanPopulateAndValidateCompositeConfig() {
	// Define test values
	testValues := map[string]string{
		"DB_HOST":     "test-db",
		"DB_USERNAME": "testuser",
		"DB_PASSWORD": "testpass",
		"DB_PORT":     "5433",
		"REDIS_HOST":  "test-redis",
		"REDIS_PORT":  "6380",
	}

	expectedAppName := "test-app"
	expectedDBPort := 5433 // Port gets converted to 8080 in Populate logic
	expectedRedisPort := 6380

	// Setup environment variables
	for key, value := range testValues {
		_ = os.Setenv(key, value)
	}
	defer func() {
		for key := range testValues {
			_ = os.Unsetenv(key)
		}
	}()

	compositeConfig := NewCompositeConfig()
	appConfig := &AppConfig{
		AppName: expectedAppName,
	}

	err := compositeConfig.PopulateAndValidate(appConfig, "test", ".")

	suite.Assert().NoError(err)
	suite.Assert().Equal(testValues["DB_HOST"], appConfig.Database.Host)
	suite.Assert().Equal(testValues["DB_USERNAME"], appConfig.Database.Username)
	suite.Assert().Equal(testValues["DB_PASSWORD"], appConfig.Database.Password)
	suite.Assert().Equal(expectedDBPort, appConfig.Database.Port)
	suite.Assert().Equal(testValues["REDIS_HOST"], appConfig.Redis.Host)
	suite.Assert().Equal(expectedRedisPort, appConfig.Redis.Port)
	suite.Assert().Equal(expectedAppName, appConfig.AppName)
}

// TestItFailsValidationWhenRequired tests validation failure
func (suite *ConfigTestSuite) TestItFailsValidationWhenRequired() {
	compositeConfig := NewCompositeConfig()
	appConfig := &AppConfig{
		// Missing required AppName
	}

	err := compositeConfig.PopulateAndValidate(appConfig, "test", ".")

	suite.Assert().Error(err)
	suite.Assert().Contains(err.Error(), "validation failed")
}

// TestItHandlesPopulateErrors tests handling of populate() errors
func (suite *ConfigTestSuite) TestItHandlesPopulateErrors() {
	compositeConfig := NewCompositeConfig()
	failingComposite := &CompositeWithFailingConfig{}

	err := compositeConfig.PopulateAndValidate(failingComposite, "test", ".")

	suite.Assert().Error(err)
	suite.Assert().Contains(err.Error(), "failed to populate nested configs")
	suite.Assert().Contains(err.Error(), "populate failed")
}

// TestItRejectsNonStructTypes tests error handling for invalid types
func (suite *ConfigTestSuite) TestItRejectsNonStructTypes() {
	compositeConfig := NewCompositeConfig()
	invalidInput := "not a struct"

	err := compositeConfig.PopulateAndValidate(invalidInput, "test", ".")

	suite.Assert().Error(err)
	suite.Assert().Contains(err.Error(), "expected struct or pointer to struct")
}

// TestItCanLoadEnvVars tests the existing LoadEnvVars function
func (suite *ConfigTestSuite) TestItCanLoadEnvVars() {
	// Define test values
	testVarName := "TEST_VAR"
	testVarValue := "test_value"
	testEnv := "test"

	tempDir := suite.T().TempDir()

	// Create a test .env file
	envFile := tempDir + "\\.env." + testEnv
	envContent := testVarName + "=" + testVarValue
	err := os.WriteFile(envFile, []byte(envContent), 0644)
	suite.Assert().NoError(err)

	err = LoadEnvVars(testEnv, tempDir)
	suite.Assert().NoError(err)

	// Check if the variable was loaded
	value := os.Getenv(testVarName)
	suite.Assert().Equal(testVarValue, value)

	// Cleanup
	_ = os.Unsetenv(testVarName)
}

// TestItHandlesNonExistentEnvFiles tests LoadEnvVars with non-existent files
func (suite *ConfigTestSuite) TestItHandlesNonExistentEnvFiles() {
	// Define test values
	nonExistentEnv := "nonexistent"

	tempDir := suite.T().TempDir()

	err := LoadEnvVars(nonExistentEnv, tempDir)
	suite.Assert().NoError(err) // Should not error when files don't exist
}

// TestConfigDebugStringConfig is a test struct with various field types for testing Debug
type TestConfigDebugStringConfig struct {
	Host        string
	Port        int
	Password    string
	DatabaseDSN string
	SecretKey   string
	APIKey      string
	Debug       bool
}

// NestedTestConfig represents a nested configuration for testing
type NestedTestConfig struct {
	Database TestConfigDebugStringConfig
	Redis    struct {
		Host     string
		Password string
	}
	Tags []string
	Meta map[string]string
}

// TestItCanDebugConfigStringWithSensitiveMasking tests Debug with sensitive field masking
func (suite *ConfigTestSuite) TestItCanDebugConfigStringWithSensitiveMasking() {
	config := TestConfigDebugStringConfig{
		Host:        "localhost",
		Port:        5432,
		Password:    "secretpassword",
		DatabaseDSN: "postgres://user:pass@localhost/db",
		SecretKey:   "mysecretkey",
		APIKey:      "myapikey",
		Debug:       true,
	}

	sensitiveKeys := []string{"pass", "secret", "key", "dsn"}
	result := Debug(config, sensitiveKeys)

	suite.Assert().Contains(result, "Config Debug Output:")
	suite.Assert().Contains(result, "Host: localhost")
	suite.Assert().Contains(result, "Port: 5432")
	suite.Assert().Contains(result, "Debug: true")

	// Sensitive fields should be masked using maskSensitiveData function
	suite.Assert().Contains(result, "Password: s************d")
	suite.Assert().Contains(result, "DatabaseDSN: p******************s@localhost/db")
	suite.Assert().Contains(result, "SecretKey: m*********y")
	suite.Assert().Contains(result, "APIKey: m******y")

	// Ensure actual sensitive values are not in the output
	suite.Assert().NotContains(result, "secretpassword")
	suite.Assert().NotContains(result, "postgres://user:pass@localhost/db")
	suite.Assert().NotContains(result, "mysecretkey")
	suite.Assert().NotContains(result, "myapikey")
}

// TestItCanDebugConfigStringWithNestedStructs tests Debug with nested structures
func (suite *ConfigTestSuite) TestItCanDebugConfigStringWithNestedStructs() {
	config := NestedTestConfig{
		Database: TestConfigDebugStringConfig{
			Host:     "db-host",
			Port:     5432,
			Password: "dbpassword",
		},
		Redis: struct {
			Host     string
			Password string
		}{
			Host:     "redis-host",
			Password: "redispassword",
		},
		Tags: []string{"prod", "db", "cache"},
		Meta: map[string]string{
			"version":    "1.0",
			"secret_env": "prod-secret",
		},
	}

	sensitiveKeys := []string{"pass", "secret"}
	result := Debug(config, sensitiveKeys)

	suite.Assert().Contains(result, "Config Debug Output:")
	suite.Assert().Contains(result, "Database:")
	suite.Assert().Contains(result, "Host: db-host")
	suite.Assert().Contains(result, "Port: 5432")
	suite.Assert().Contains(result, "Password: d********d")
	suite.Assert().Contains(result, "Redis:")
	suite.Assert().Contains(result, "Host: redis-host")
	suite.Assert().Contains(result, "Password: r***********d")
	suite.Assert().Contains(result, "Tags:")
	suite.Assert().Contains(result, "[0]: prod")
	suite.Assert().Contains(result, "[1]: db")
	suite.Assert().Contains(result, "[2]: cache")
	suite.Assert().Contains(result, "Meta:")
	suite.Assert().Contains(result, "version: 1.0")
	suite.Assert().Contains(result, "secret_env: p*********t")

	// Ensure sensitive values are not exposed
	suite.Assert().NotContains(result, "dbpassword")
	suite.Assert().NotContains(result, "redispassword")
	suite.Assert().NotContains(result, "prod-secret")
}

// TestItCanDebugConfigStringWithNilValues tests Debug with nil values
func (suite *ConfigTestSuite) TestItCanDebugConfigStringWithNilValues() {
	var config *TestConfigDebugStringConfig = nil
	sensitiveKeys := []string{"pass", "secret"}

	result := Debug(config, sensitiveKeys)

	suite.Assert().Equal("Config Debug Output:\nnil\n", result)
}

// TestItCanDebugConfigStringWithEmptyStruct tests Debug with empty struct
func (suite *ConfigTestSuite) TestItCanDebugConfigStringWithEmptyStruct() {
	config := struct{}{}
	sensitiveKeys := []string{"pass", "secret"}

	result := Debug(config, sensitiveKeys)

	suite.Assert().Contains(result, "Config Debug Output:")
	// Empty struct should not have any field entries (lines with indentation and field names)
	lines := strings.Split(result, "\n")
	fieldLines := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") { // Lines with indentation indicate fields
			fieldLines++
		}
	}
	suite.Assert().Equal(0, fieldLines, "Empty struct should have no field entries")
}

// TestItCanDebugConfigStringWithEmptySliceAndMap tests Debug with empty collections
func (suite *ConfigTestSuite) TestItCanDebugConfigStringWithEmptySliceAndMap() {
	config := struct {
		EmptySlice []string
		EmptyMap   map[string]string
	}{
		EmptySlice: []string{},
		EmptyMap:   map[string]string{},
	}

	sensitiveKeys := []string{"pass", "secret"}
	result := Debug(config, sensitiveKeys)

	suite.Assert().Contains(result, "Config Debug Output:")
	suite.Assert().Contains(result, "EmptySlice:")
	suite.Assert().Contains(result, "[]")
	suite.Assert().Contains(result, "EmptyMap:")
	suite.Assert().Contains(result, "{}")
}

// TestItCanDebugConfigStringWithCaseInsensitiveSensitiveKeys tests case-insensitive matching
func (suite *ConfigTestSuite) TestItCanDebugConfigStringWithCaseInsensitiveSensitiveKeys() {
	config := struct {
		PASSWORD   string
		ApiKey     string
		secretHash string
		DSN        string
	}{
		PASSWORD:   "mypass",
		ApiKey:     "mykey",
		secretHash: "hash123",
		DSN:        "connection-string",
	}

	sensitiveKeys := []string{"PASS", "key", "Secret", "dsn"}
	result := Debug(config, sensitiveKeys)

	// All should be masked due to case-insensitive matching
	suite.Assert().Contains(result, "PASSWORD: m****s")
	suite.Assert().Contains(result, "ApiKey: m***y")
	suite.Assert().Contains(result, "DSN: c***************g")

	// Ensure actual values are not exposed
	suite.Assert().NotContains(result, "mypass")
	suite.Assert().NotContains(result, "mykey")
	suite.Assert().NotContains(result, "hash123")
	suite.Assert().NotContains(result, "connection-string")
}

// TestItCanDebugConfigStringWithNoSensitiveKeys tests Debug without sensitive keys
func (suite *ConfigTestSuite) TestItCanDebugConfigStringWithNoSensitiveKeys() {
	config := TestConfigDebugStringConfig{
		Host:     "localhost",
		Port:     5432,
		Password: "password123",
		Debug:    false,
	}

	var sensitiveKeys []string // No sensitive keys
	result := Debug(config, sensitiveKeys)

	suite.Assert().Contains(result, "Config Debug Output:")
	suite.Assert().Contains(result, "Host: localhost")
	suite.Assert().Contains(result, "Port: 5432")
	suite.Assert().Contains(result, "Password: password123") // Should not be masked
	suite.Assert().Contains(result, "Debug: false")
}

// Run the test suite
func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
