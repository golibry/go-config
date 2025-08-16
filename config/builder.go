package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

type Config interface {
	Populate() error
}

// CompositeConfig represents a configuration that contains nested config structs.
// It automatically populates and validates all nested structs that implement the Config interface.
type CompositeConfig struct {
	validator *validator.Validate
}

// NewCompositeConfig creates a new CompositeConfig with a validator instance.
func NewCompositeConfig() *CompositeConfig {
	return &CompositeConfig{
		validator: validator.New(),
	}
}

// PopulateAndValidate populates all nested Config structs and validates the composite struct.
// It uses reflection to find all struct fields that implement the Config interface,
// calls their Populate() method, and then validates the entire composite struct.
func (c *CompositeConfig) PopulateAndValidate(
	compositeStruct interface{},
	defaultEnv string,
	defaultAppDir string,
) error {
	// Load environment variables first
	if err := LoadEnvVars(defaultEnv, defaultAppDir); err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}

	if err := c.populateNestedConfigs(compositeStruct); err != nil {
		return fmt.Errorf("failed to populate nested configs: %w", err)
	}

	if err := c.validator.Struct(compositeStruct); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// populateNestedConfigs uses reflection to find and populate all nested Config structs.
func (c *CompositeConfig) populateNestedConfigs(compositeStruct interface{}) error {
	val := reflect.ValueOf(compositeStruct)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct or pointer to struct, got %T", compositeStruct)
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Check if field implements Config interface
		if c.implementsConfig(field) {
			if err := c.callPopulate(field); err != nil {
				return fmt.Errorf("failed to populate field %s: %w", fieldType.Name, err)
			}
		}

		// Recursively handle embedded structs
		if field.Kind() == reflect.Struct {
			if err := c.populateNestedConfigs(field.Addr().Interface()); err != nil {
				return fmt.Errorf(
					"failed to populate nested struct in field %s: %w",
					fieldType.Name,
					err,
				)
			}
		}
	}

	return nil
}

// implementsConfig checks if a reflect.Value implements the Config interface.
func (c *CompositeConfig) implementsConfig(val reflect.Value) bool {
	if !val.CanInterface() {
		return false
	}

	configType := reflect.TypeOf((*Config)(nil)).Elem()
	return val.Type().Implements(configType) ||
		(val.CanAddr() && val.Addr().Type().Implements(configType))
}

// callPopulate calls the Populate method on a Config interface.
func (c *CompositeConfig) callPopulate(val reflect.Value) error {
	var config Config

	if val.Type().Implements(reflect.TypeOf((*Config)(nil)).Elem()) {
		config = val.Interface().(Config)
	} else if val.CanAddr() && val.Addr().Type().Implements(reflect.TypeOf((*Config)(nil)).Elem()) {
		config = val.Addr().Interface().(Config)
	} else {
		return fmt.Errorf("field does not implement Config interface")
	}

	return config.Populate()
}

// LoadEnvVars Loads the entries from env files and sets them as env variables for this process.
// Loads each file in order: .env.{dev|prod|test}.local, .env.local, .env.{dev|prod|test}, .env.
// The first loaded file has priority. Files will not overwrite the values of the already loaded
// env vars (already loaded from env files or via other means).
func LoadEnvVars(env string, appBaseDir string) error {
	localEnvFileName := filepath.Join(appBaseDir, ".env."+env+".local")
	if _, err := os.Stat(localEnvFileName); err == nil {
		err := godotenv.Load(localEnvFileName)

		if err != nil {
			return formatEnvLoadErr(localEnvFileName, err)
		}
	}

	genericLocalFileName := filepath.Join(appBaseDir, ".env.local")

	if env != "test" {
		if _, err := os.Stat(genericLocalFileName); err == nil {
			err := godotenv.Load(genericLocalFileName)

			if err != nil {
				return formatEnvLoadErr(genericLocalFileName, err)
			}
		}
	}

	genericEnvFileName := filepath.Join(appBaseDir, ".env."+env)
	if _, err := os.Stat(genericEnvFileName); err == nil {
		err := godotenv.Load(genericEnvFileName)

		if err != nil {
			return formatEnvLoadErr(genericEnvFileName, err)
		}
	}

	baseEnvFileName := filepath.Join(appBaseDir, ".env")
	if _, err := os.Stat(baseEnvFileName); err == nil {
		err := godotenv.Load(baseEnvFileName)

		if err != nil {
			return formatEnvLoadErr(baseEnvFileName, err)
		}
	}

	return nil
}

func formatEnvLoadErr(fileName string, err error) error {
	return fmt.Errorf(
		"error occurred while trying to load env file: %s. Error message: %s",
		fileName,
		err.Error(),
	)
}
