package db

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
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
		driver = "mysql"
	}

	return Config{
		Driver:   driver,
		Host:     getEnvWithDefault("DB_HOST", "localhost"),
		Port:     getEnvWithDefault("DB_PORT", "3306"),
		User:     getEnvWithDefault("DB_USER", "root"),
		Password: getEnvWithDefault("DB_PASSWORD", "password"),
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
	var dialector gorm.Dialector
	var err error

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	if config.Driver == "sqlite" {
		dsn := config.Database
		if dsn == "" {
			dsn = ":memory:"
		}
		dialector = sqlite.Open(dsn)
	} else if config.Driver == "mysql" {
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
			config.User, config.Password, config.Host, config.Port, config.Database,
		)
		dialector = mysql.Open(dsn)
	} else {
		return fmt.Errorf("unsupported driver: %s", config.Driver)
	}

	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	currentDriver = config.Driver

	return nil
}

func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func IsSQLite() bool {
	return currentDriver == "sqlite"
}
