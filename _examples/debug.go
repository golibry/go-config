package main

import (
	"fmt"

	"github.com/golibry/go-config/config"
)

// ExampleConfig demonstrates the Debug function
type ExampleConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	DatabaseDSN string
	APIKey      string
	SecretToken string
	Debug       bool
}

// NestedConfig shows nested structure debugging
type NestedConfig struct {
	Database ExampleConfig
	Redis    struct {
		Host     string
		Password string
	}
	Settings map[string]string
	Tags     []string
}

func main() {
	fmt.Println("Debug Function Demonstration")
	fmt.Println("=======================================")

	// Example 1: Simple config with sensitive fields
	fmt.Println("\n1. Simple Configuration with Sensitive Fields:")
	simpleConfig := ExampleConfig{
		Host:        "localhost",
		Port:        5432,
		Username:    "admin",
		Password:    "supersecret123",
		DatabaseDSN: "postgres://user:pass@localhost:5432/mydb",
		APIKey:      "api_key_abc123",
		SecretToken: "secret_token_xyz789",
		Debug:       true,
	}

	sensitiveKeys := []string{"pass", "secret", "key", "dsn"}
	debugOutput := config.Debug(simpleConfig, sensitiveKeys)
	fmt.Print(debugOutput)

	// Example 2: Nested configuration
	fmt.Println("\n2. Nested Configuration:")
	nestedConfig := NestedConfig{
		Database: ExampleConfig{
			Host:     "db.example.com",
			Port:     5432,
			Username: "dbuser",
			Password: "dbpassword123",
		},
		Redis: struct {
			Host     string
			Password string
		}{
			Host:     "redis.example.com",
			Password: "redispassword456",
		},
		Settings: map[string]string{
			"env":        "production",
			"secret_key": "my_secret_setting",
			"timeout":    "30s",
		},
		Tags: []string{"production", "database", "cache"},
	}

	debugOutput2 := config.Debug(nestedConfig, sensitiveKeys)
	fmt.Print(debugOutput2)

	// Example 3: No sensitive keys (shows all values)
	fmt.Println("\n3. Configuration without Sensitive Key Masking:")
	debugOutput3 := config.Debug(simpleConfig, []string{})
	fmt.Print(debugOutput3)

	// Example 4: Nil configuration
	fmt.Println("\n4. Nil Configuration:")
	var nilConfig *ExampleConfig = nil
	debugOutput4 := config.Debug(nilConfig, sensitiveKeys)
	fmt.Print(debugOutput4)

	fmt.Println("\n✅ Debug demonstration completed!")
}
