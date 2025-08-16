package config

import (
	"errors"
	"os"
	"strconv"
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

// Run the test suite
func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
