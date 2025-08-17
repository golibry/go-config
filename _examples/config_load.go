package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golibry/go-config/config"
)

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string `env:"DB_HOST" validate:"required"`
	Port     int    `env:"DB_PORT" validate:"min=1,max=65535"`
	Username string `env:"DB_USERNAME" validate:"required"`
	Password string `env:"DB_PASSWORD" validate:"required"`
	Database string `env:"DB_NAME" validate:"required"`
}

// Populate implements the Config interface for DatabaseConfig
func (d *DatabaseConfig) Populate() error {
	d.Host = getEnvOrDefault("DB_HOST", "localhost")
	d.Username = getEnvOrDefault("DB_USERNAME", "admin")
	d.Password = getEnvOrDefault("DB_PASSWORD", "password")
	d.Database = getEnvOrDefault("DB_NAME", "myapp")

	// Parse port from environment or use default
	portStr := os.Getenv("DB_PORT")
	if portStr == "" {
		d.Port = 5432
	} else {
		// In a real implementation, you'd parse the string to int
		// For simplicity, using a fixed value here
		d.Port = 5433
	}

	return nil
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string `env:"REDIS_HOST" validate:"required"`
	Port     int    `env:"REDIS_PORT" validate:"min=1,max=65535"`
	Password string `env:"REDIS_PASSWORD"`
}

// Populate implements the Config interface for RedisConfig
func (r *RedisConfig) Populate() error {
	r.Host = getEnvOrDefault("REDIS_HOST", "localhost")
	r.Password = os.Getenv("REDIS_PASSWORD") // Optional

	// Parse port from environment or use default
	portStr := os.Getenv("REDIS_PORT")
	if portStr == "" {
		r.Port = 6379
	} else {
		// In a real implementation, you'd parse the string to int
		// For simplicity, using a fixed value here
		r.Port = 6380
	}

	return nil
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string `env:"SERVER_HOST" validate:"required"`
	Port int    `env:"SERVER_PORT" validate:"min=1,max=65535"`
}

// Populate implements the Config interface for ServerConfig
func (s *ServerConfig) Populate() error {
	s.Host = getEnvOrDefault("SERVER_HOST", "0.0.0.0")

	// Parse port from environment or use default
	portStr := os.Getenv("SERVER_PORT")
	if portStr == "" {
		s.Port = 8080
	} else {
		// In a real implementation, you'd parse the string to int
		// For simplicity, using a fixed value here
		s.Port = 8081
	}

	return nil
}

// AppConfig is the main composite configuration struct
type AppConfig struct {
	Database DatabaseConfig `validate:"required"`
	Redis    RedisConfig    `validate:"required"`
	Server   ServerConfig   `validate:"required"`
	AppName  string         `env:"APP_NAME" validate:"required"`
	Debug    bool           `env:"DEBUG"`
}

// Populate implements the Config interface for AppConfig
func (a *AppConfig) Populate() error {
	a.AppName = getEnvOrDefault("APP_NAME", "DefaultApp")

	// Parse debug from environment or use default
	debugStr := os.Getenv("DEBUG")
	if debugStr == "true" || debugStr == "1" {
		a.Debug = true
	} else {
		a.Debug = false
	}

	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	fmt.Println("Go-Config Composite Configuration Example")
	fmt.Println("========================================")

	// Set some example environment variables for demonstration
	_ = os.Setenv("APP_NAME", "MyAwesomeApp")
	_ = os.Setenv("DEBUG", "true")
	_ = os.Setenv("DB_HOST", "postgres.example.com")
	_ = os.Setenv("DB_USERNAME", "myuser")
	_ = os.Setenv("DB_PASSWORD", "mypassword")
	_ = os.Setenv("DB_NAME", "production_db")
	_ = os.Setenv("REDIS_HOST", "redis.example.com")
	_ = os.Setenv("SERVER_PORT", "3000")

	// Create composite config instance
	compositeConfig := config.NewCompositeConfig()

	// Create your application config struct
	appConfig := &AppConfig{}

	// Manually populate the top-level config first
	err := appConfig.Populate()
	if err != nil {
		log.Fatalf("Failed to populate app config: %v", err)
	}

	// Populate and validate all nested configurations
	// LoadEnvVars is automatically called within PopulateAndValidate
	err = compositeConfig.PopulateAndValidate(appConfig, "dev", ".")
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Display the populated configuration
	fmt.Println("\nConfiguration loaded successfully!")
	fmt.Printf("App Name: %s\n", appConfig.AppName)
	fmt.Printf("Debug Mode: %t\n", appConfig.Debug)

	fmt.Println("\nDatabase Configuration:")
	fmt.Printf("  Host: %s\n", appConfig.Database.Host)
	fmt.Printf("  Port: %d\n", appConfig.Database.Port)
	fmt.Printf("  Username: %s\n", appConfig.Database.Username)
	fmt.Printf("  Password: %s\n", maskPassword(appConfig.Database.Password))
	fmt.Printf("  Database: %s\n", appConfig.Database.Database)

	fmt.Println("\nRedis Configuration:")
	fmt.Printf("  Host: %s\n", appConfig.Redis.Host)
	fmt.Printf("  Port: %d\n", appConfig.Redis.Port)
	fmt.Printf("  Password: %s\n", maskPassword(appConfig.Redis.Password))

	fmt.Println("\nServer Configuration:")
	fmt.Printf("  Host: %s\n", appConfig.Server.Host)
	fmt.Printf("  Port: %d\n", appConfig.Server.Port)

	fmt.Println("\n✅ All configurations populated and validated successfully!")
}

// maskPassword masks password for display purposes
func maskPassword(password string) string {
	if password == "" {
		return "(not set)"
	}
	if len(password) <= 2 {
		return "***"
	}
	return password[:2] + "***"
}
