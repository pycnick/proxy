package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pycnick/proxy/internal/database/postgres/connector"
	"github.com/pycnick/proxy/internal/proxy/delivery"
	"github.com/pycnick/proxy/internal/proxy/repository"
	"github.com/pycnick/proxy/internal/proxy/usecase"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	port := os.Getenv("PORT")
	if port == "" {
		logger.Debug("No PORT env...")
		port = "8080"
	}

	port = ":" + port

	connector := connector.NewPostgresConnector()
	connPool, err := connector.Connect()
	if err != nil {
		fmt.Println(err)
		logger.Debug(err)
		return
	}

	proxyServer := echo.New()

	proxyServer.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	personRepository := repository.NewProxyRepository(logger, connPool)
	personsUseCase, _ := usecase.NewProxyUseCase(logger, personRepository)
	_ = delivery.NewHttpDelivery(proxyServer, logger, personsUseCase)

	logger.Debug(proxyServer.Start(port))
}
