package config

import (
	"encoding/json"
	"log"
	"os"
)

var Appconfig *AppConfig

const configPath = "./config.json"

func Load() error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Config file %s not found. Using default configuration.")
		Appconfig = defaultConfig()
		return nil
	}

	configFile, err := os.Open("./config.json")
	defer configFile.Close()
	if err != nil {
		log.Println("config file doesn't exists")
		return err
	}

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&Appconfig); err != nil {
		return err
	}

	return nil
}

func defaultConfig() *AppConfig {
	return &AppConfig{
		Postgres: PostgresConfig{
			Host:     "postgres",
			Port:     "5432",
			DBName:   "sse",
			User:     "postgres",
			Password: "postgres",
		},
		HTTPServer: HTTPServerConfig{
			Port: "8080",
		},
	}
}
