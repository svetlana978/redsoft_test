package config

import (
	"flag"
	"fmt"
	"time"
)

type Config struct {
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration

	GRPCGracefulTimeout time.Duration
	GRPCPort            string
	LogLevel            string

	EnrichTimeout  time.Duration
	AllowTimeout   time.Duration
	AgifyURL       string
	GenderizeURL   string
	NationalizeURL string
}

func Load() *Config {
	config := &Config{}

	flag.StringVar(&config.DBHost, "db_host", "localhost", "PostgreSQL host")
	flag.IntVar(&config.DBPort, "db_port", 5433, "PostgreSQL port")
	flag.StringVar(&config.DBUser, "db_user", "postgres", "PostgreSQL user")
	flag.StringVar(&config.DBPassword, "db_password", "test_password", "PostgreSQL password")
	flag.StringVar(&config.DBName, "db_name", "redsoft2", "PostgreSQL database name")
	flag.StringVar(&config.DBSSLMode, "db_sslmode", "disable", "PostgreSQL SSL mode")

	flag.IntVar(&config.DBMaxOpenConns, "db_max_open_conns", 50, "Maximum number of open connections")
	flag.IntVar(&config.DBMaxIdleConns, "db_max_idle_conns", 25, "Maximum number of idle connections")
	flag.DurationVar(&config.DBConnMaxLifetime, "db_conn_max_lifetime", 30*time.Minute, "Maximum connection lifetime")
	flag.DurationVar(&config.DBConnMaxIdleTime, "db_conn_max_idle_time", 10*time.Minute, "Maximum idle connection time")

	flag.DurationVar(&config.GRPCGracefulTimeout, "grpc_graceful_timeout", 30*time.Second, "gRPC server graceful timeout")
	flag.StringVar(&config.GRPCPort, "grpc_port", "50051", "gRPC server port")
	flag.StringVar(&config.LogLevel, "log_level", "info", "Log level (debug, info, warn, error)")

	flag.DurationVar(&config.EnrichTimeout, "enrich_timeout", 5*time.Second, "Timeout for external API calls")
	flag.DurationVar(&config.AllowTimeout, "allow_timeout", 1*time.Minute, "external API request rate limiting timedout")
	flag.StringVar(&config.AgifyURL, "agify_url", "https://api.agify.io", "Agify API URL")
	flag.StringVar(&config.GenderizeURL, "genderize_url", "https://api.genderize.io", "Genderize API URL")
	flag.StringVar(&config.NationalizeURL, "nationalize_url", "https://api.nationalize.io", "Nationalize API URL")

	flag.Parse()
	return config
}

func (c *Config) GetDBConnString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}
