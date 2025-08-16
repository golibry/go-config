# go-config

A flexible Go library for building and composing configuration for web applications. The library provides a clean, structured approach to handle environment-based configuration with validation support.

## Features

- **Environment-based configuration**: Supports multiple `.env` files with priority ordering
- **Composite configuration**: Automatically populate and validate nested configuration structs
- **Validation support**: Built-in validation using `go-playground/validator`
- **Flexible structure**: Implement the `Config` interface for custom configuration logic
- **Zero dependencies**: Minimal external dependencies for core functionality

## Installation

Add the library to your Go project:

```bash
go get github.com/golibry/go-config
```

## Quick Start

### 1. Define your configuration structs

```go
package main

import (
    "os"
    "github.com/golibry/go-config/config"
)

// DatabaseConfig implements the Config interface
type DatabaseConfig struct {
    Host     string `validate:"required"`
    Port     int    `validate:"min=1,max=65535"`
    Username string `validate:"required"`
    Password string `validate:"required"`
}

func (d *DatabaseConfig) Populate() error {
    d.Host = os.Getenv("DB_HOST")
    d.Username = os.Getenv("DB_USERNAME")
    d.Password = os.Getenv("DB_PASSWORD")
    // Add your custom population logic here
    return nil
}

// AppConfig is your main composite configuration
type AppConfig struct {
    Database DatabaseConfig `validate:"required"`
    AppName  string         `validate:"required"`
    Debug    bool
}

func (a *AppConfig) Populate() error {
    a.AppName = os.Getenv("APP_NAME")
    a.Debug = os.Getenv("DEBUG") == "true"
    return nil
}
```

### 2. Load and validate configuration

```go
func main() {
    // Create composite config instance
    compositeConfig := config.NewCompositeConfig()
    
    // Create your application config
    appConfig := &AppConfig{}
    
    // Populate top-level config
    err := appConfig.Populate()
    if err != nil {
        log.Fatalf("Failed to populate app config: %v", err)
    }
    
    // Populate nested configs and validate
    err = compositeConfig.PopulateAndValidate(appConfig, "dev", ".")
    if err != nil {
        log.Fatalf("Configuration error: %v", err)
    }
    
    // Your configuration is now ready to use
    fmt.Printf("App: %s, DB Host: %s\n", appConfig.AppName, appConfig.Database.Host)
}
```

## Environment File Priority

The library loads environment files in the following order (first found takes priority):

1. `.env.{env}.local` (e.g., `.env.dev.local`)
2. `.env.local` (not loaded in test environment)
3. `.env.{env}` (e.g., `.env.dev`)
4. `.env`

## Configuration Interface

Implement the `Config` interface for any struct that needs custom population logic:

```go
type Config interface {
    Populate() error
}
```

## Validation

The library uses `go-playground/validator` for struct validation. Add validation tags to your struct fields:

```go
type ServerConfig struct {
    Host string `validate:"required"`
    Port int    `validate:"min=1,max=65535"`
}
```

## Examples

For complete working examples, see the [`_examples`](_examples/) directory:

- **[Basic Example](_examples/main.go)**: Demonstrates composite configuration with database, Redis, and server configs

Run the example:

```bash
cd _examples
go run main.go
```

## Requirements

- Go 1.21 or later

## Dependencies

- `github.com/go-playground/validator/v10` - Struct validation
- `github.com/joho/godotenv` - Environment file loading

## License

This project is licensed under the terms specified in the LICENSE file.
