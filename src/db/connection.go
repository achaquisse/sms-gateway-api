package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

var DB *sql.DB
var currentDriver string

type Config struct {
	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func GetConfigFromEnv() Config {
	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		driver = "pgx"
	}

	return Config{
		Driver:   driver,
		Host:     getEnvWithDefault("DB_HOST", "localhost"),
		Port:     getEnvWithDefault("DB_PORT", "5432"),
		User:     getEnvWithDefault("DB_USER", "postgres"),
		Password: getEnvWithDefault("DB_PASSWORD", "postgres"),
		Database: getEnvWithDefault("DB_NAME", "sms_gateway"),
		SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func Connect() error {
	config := GetConfigFromEnv()
	return ConnectWithConfig(config)
}

func ConnectWithConfig(config Config) error {
	var dsn string
	var err error

	if config.Driver == "sqlite" {
		dsn = config.Database
		if dsn == "" {
			dsn = ":memory:"
		}
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
		)
	}

	DB, err = sql.Open(config.Driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	currentDriver = config.Driver

	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func GetDB() *sql.DB {
	return DB
}

func IsSQLite() bool {
	return currentDriver == "sqlite"
}
