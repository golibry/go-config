package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

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
func NewCompositeConfig(customValidator *validator.Validate) *CompositeConfig {
	if customValidator == nil {
		customValidator = validator.New()
	}
	return &CompositeConfig{
		validator: customValidator,
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

// Debug transforms a config struct recursively into a string for debugging.
// Sensitive attributes (matching keywords in sensitiveKeys) are masked with "***".
// The sensitiveKeys slice contains keywords to check against field names (case-insensitive).
func Debug(config interface{}, sensitiveKeys []string) string {
	if config == nil {
		return "nil"
	}

	var result strings.Builder
	result.WriteString("Config Debug Output:\n")
	debugValue(reflect.ValueOf(config), sensitiveKeys, &result, 0)
	return result.String()
}

// debugValue recursively processes a reflect.Value and builds the debug string
func debugValue(val reflect.Value, sensitiveKeys []string, builder *strings.Builder, indent int) {
	// Handle pointers by dereferencing them
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			writeIndent(builder, indent)
			builder.WriteString("nil\n")
			return
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		debugStruct(val, sensitiveKeys, builder, indent)
	case reflect.Slice, reflect.Array:
		debugSlice(val, sensitiveKeys, builder, indent)
	case reflect.Map:
		debugMap(val, sensitiveKeys, builder, indent)
	default:
		writeIndent(builder, indent)
		builder.WriteString(fmt.Sprintf("%v\n", val.Interface()))
	}
}

// debugStruct processes struct fields recursively
func debugStruct(val reflect.Value, sensitiveKeys []string, builder *strings.Builder, indent int) {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		writeIndent(builder, indent)
		fieldName := fieldType.Name
		builder.WriteString(fmt.Sprintf("%s: ", fieldName))

		// Check if this field name matches any sensitive keywords
		if isSensitiveField(fieldName, sensitiveKeys) {
			fieldValue := fmt.Sprintf("%v", field.Interface())
			maskedValue := maskSensitiveData(fieldValue)
			builder.WriteString(maskedValue + "\n")
			continue
		}

		if field.Kind() == reflect.Struct ||
			field.Kind() == reflect.Slice ||
			field.Kind() == reflect.Array ||
			field.Kind() == reflect.Map {
			builder.WriteString("\n")
			debugValue(field, sensitiveKeys, builder, indent+1)
		} else {
			builder.WriteString(fmt.Sprintf("%v\n", field.Interface()))
		}
	}
}

// debugSlice processes slice/array elements
func debugSlice(val reflect.Value, sensitiveKeys []string, builder *strings.Builder, indent int) {
	length := val.Len()
	if length == 0 {
		writeIndent(builder, indent)
		builder.WriteString("[]\n")
		return
	}

	for i := 0; i < length; i++ {
		writeIndent(builder, indent)
		builder.WriteString(fmt.Sprintf("[%d]: ", i))

		elem := val.Index(i)
		if elem.Kind() == reflect.Struct {
			builder.WriteString("\n")
			debugValue(elem, sensitiveKeys, builder, indent+1)
		} else {
			builder.WriteString(fmt.Sprintf("%v\n", elem.Interface()))
		}
	}
}

// debugMap processes map key-value pairs
func debugMap(val reflect.Value, sensitiveKeys []string, builder *strings.Builder, indent int) {
	keys := val.MapKeys()
	if len(keys) == 0 {
		writeIndent(builder, indent)
		builder.WriteString("{}\n")
		return
	}

	for _, key := range keys {
		writeIndent(builder, indent)
		keyStr := fmt.Sprintf("%v", key.Interface())
		builder.WriteString(fmt.Sprintf("%s: ", keyStr))

		mapVal := val.MapIndex(key)

		// Check if this key matches any sensitive keywords
		if isSensitiveField(keyStr, sensitiveKeys) {
			mapValue := fmt.Sprintf("%v", mapVal.Interface())
			maskedValue := maskSensitiveData(mapValue)
			builder.WriteString(maskedValue + "\n")
			continue
		}
		if mapVal.Kind() == reflect.Struct {
			builder.WriteString("\n")
			debugValue(mapVal, sensitiveKeys, builder, indent+1)
		} else {
			builder.WriteString(fmt.Sprintf("%v\n", mapVal.Interface()))
		}
	}
}

// isSensitiveField checks if a field name matches any sensitive keywords (case-insensitive)
func isSensitiveField(fieldName string, sensitiveKeys []string) bool {
	lowerFieldName := strings.ToLower(fieldName)

	for _, sensitiveKey := range sensitiveKeys {
		if strings.Contains(lowerFieldName, strings.ToLower(sensitiveKey)) {
			return true
		}
	}

	return false
}

// writeIndent writes the appropriate indentation to the builder
func writeIndent(builder *strings.Builder, indent int) {
	for i := 0; i < indent; i++ {
		builder.WriteString("  ")
	}
}

// maskSensitiveData masks sensitive information in a string for safe logging.
func maskSensitiveData(data string) string {
	if data == "" {
		return ""
	}

	// For DSN strings, mask everything after the first character and before the last character
	if strings.Contains(data, "@") && strings.Contains(data, ":") {
		parts := strings.Split(data, "@")
		if len(parts) >= 2 {
			// Mask the user:password part
			userPass := parts[0]
			if len(userPass) > 2 {
				masked := string(userPass[0]) + strings.Repeat(
					"*",
					len(userPass)-2,
				) + string(userPass[len(userPass)-1])
				return masked + "@" + parts[1]
			}
		}
	}

	// For other sensitive data, show the first and last character with asterisks in between
	if len(data) <= 2 {
		return strings.Repeat("*", len(data))
	}

	return string(data[0]) + strings.Repeat("*", len(data)-2) + string(data[len(data)-1])
}
