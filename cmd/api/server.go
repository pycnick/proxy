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
	"sync"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		logger.Debug("No PROXY_PORT env...")
		proxyPort = "8080"
	}

	proxyPort = ":" + proxyPort

	repeaterPort := os.Getenv("REPEATER_PORT")
	if repeaterPort == "" {
		logger.Debug("No REPEATER_PORT env...")
		repeaterPort = "8081"
	}

	repeaterPort = ":" + repeaterPort

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
	personsDelivery := delivery.NewHttpDelivery(proxyServer, logger, personsUseCase)

	repeaterServer := echo.New()
	repeaterServer.POST("/repeat/:id", personsDelivery.SendRequest)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		logger.Debug(proxyServer.Start(proxyPort))
		wg.Done()
	}()

	go func() {
		logger.Debug(repeaterServer.Start(repeaterPort))
		wg.Done()
	}()

	wg.Wait()
}
