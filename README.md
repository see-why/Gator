# Gator

A Go application that demonstrates configuration management with JSON files.

## Overview

Gator is a simple Go application that showcases reading, writing, and updating JSON configuration files. It includes a custom configuration package that handles user settings and database connection information.

## Features

- **JSON Configuration Management**: Read and write configuration data to/from `.gatorconfig.json`
- **User Management**: Set and update current user information
- **Database Configuration**: Store database connection URLs
- **Clean Architecture**: Organized with internal packages following Go best practices

## Project Structure

```text
Gator/
├── .gatorconfig.json          # Configuration file
├── go.mod                     # Go module definition
├── main.go                    # Main application entry point
└── internal/
    └── config/
        └── config.go          # Configuration package
```

## Configuration File

The application uses a `.gatorconfig.json` file in the project root with the following structure:

```json
{
  "db_url": "postgres://example",
  "current_user_name": "andi"
}
```

## Getting Started

### Prerequisites

- Go 1.19 or later

### Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/see-why/Gator.git
   cd Gator
   ```

2. Ensure the configuration file exists:

   ```bash
   # The .gatorconfig.json file should already be present
   cat .gatorconfig.json
   ```

3. Run the application:

   ```bash
   go run main.go
   ```

### Usage

The application will:

1. Read the current configuration
2. Display the initial settings
3. Update the current user to "andi"
4. Save the updated configuration
5. Display the final settings

Example output:

```text
Initial config:
DB URL: postgres://example
Current User: 

Updated user to 'andi'

Final config:
DB URL: postgres://example
Current User: andi
```

## API Reference

### Config Package

The `internal/config` package provides the following exported functionality:

#### Types

```go
type Config struct {
    DbURL           string `json:"db_url"`
    CurrentUserName string `json:"current_user_name,omitempty"`
}
```

#### Functions

- `Read() (Config, error)` - Reads the configuration file and returns a Config struct
- `(cfg *Config) SetUser(username string) error` - Sets the current user and saves to file

## Development

### Building

```bash
go build -o gator main.go
```

### Testing

```bash
go test ./...
```

## License

This project is open source and available under the [MIT License](LICENSE).
