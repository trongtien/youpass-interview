package config

import (
	"errors"
	"io/fs"
	"log"
	"strconv"

	"github.com/spf13/viper"
)

type ConfigRedisEnv struct {
	Host     string
	Password string
	Db       int
}

type ConfigEnv struct {
	HttpAddr    string
	HttpPort    string
	DatabaseUrl string
	Redis       ConfigRedisEnv
}

// Used when a key is set neither in the OS environment nor in .env.
// Values match the local infra in ../infra/docker-compose.yml.
var defaultEnv = map[string]string{
	"HTTP_ADDR":      "0.0.0.0",
	"HTTP_PORT":      "8080",
	"DATABASE_URL":   "postgres://postgres:postgres@localhost:5432/share_link?sslmode=disable",
	"HOST_REDIS":     "localhost:6379",
	"PASSWORD_REDIS": "",
	"DB_REDIS":       "0",
}

func LoadConfigEnv() *ConfigEnv {
	for key, value := range defaultEnv {
		viper.SetDefault(key, value)
	}

	viper.AutomaticEnv()
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatalf("Error reading config file: %v", err)
		}
		log.Println("No .env file found, using environment variables and defaults")
	}

	dbRedis, err := strconv.Atoi(viper.GetString("DB_REDIS"))
	if err != nil {
		log.Fatalf("Error reading environment DB_REDIS: %v", err)
	}

	cfgRedis := &ConfigRedisEnv{
		Host:     viper.GetString("HOST_REDIS"),
		Password: viper.GetString("PASSWORD_REDIS"),
		Db:       dbRedis,
	}

	cfg := &ConfigEnv{
		HttpAddr:    viper.GetString("HTTP_ADDR"),
		HttpPort:    viper.GetString("HTTP_PORT"),
		DatabaseUrl: viper.GetString("DATABASE_URL"),
		Redis:       *cfgRedis,
	}

	return cfg
}
