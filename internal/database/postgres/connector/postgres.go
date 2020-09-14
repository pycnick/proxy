package connector

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
)

type PostgresConnector struct {
	Host       string `json:"host"`
	Port       string `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	Database   string `json:"database"`
	ConnString string `json:"conn_string"`
}

func NewPostgresConnector() *PostgresConnector {
	port := os.Getenv("PSQL_PORT")
	host := os.Getenv("PSQL_HOST")
	user := os.Getenv("PSQL_USER")
	password := os.Getenv("PSQL_PASS")
	database := os.Getenv("PSQL_DB")

	return &PostgresConnector{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
		ConnString: fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s",
			user, password, host, port, database),
	}
}

func (pC *PostgresConnector) Connect() (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(pC.ConnString)
	if err != nil {
		return nil, err
	}
	config.MinConns = 5
	config.MaxConns = 15

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
