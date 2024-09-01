package config

type AppConfig struct {
	Postgres   PostgresConfig   `json:"postgres"`
	HTTPServer HTTPServerConfig `json:"http_server"`
}

type PostgresConfig struct {
	Host     string `json:"host" envconfig:"POSTGRES_HOST"          default:"localhost"`
	Port     string `json:"port" envconfig:"POSTGRES_PORT"          default:"5423"`
	DBName   string `json:"db_name" envconfig:"POSTGRES_DB_NAME"          default:"sse"`
	User     string `json:"user" envconfig:"POSTGRES_USER"          default:"postgres"`
	Password string `json:"password" envconfig:"POSTGRES_PASSWORD"      default:"postgres"`
}

type HTTPServerConfig struct {
	Port string `json:"port" envconfig:"PORT" default:"8080"`
}
