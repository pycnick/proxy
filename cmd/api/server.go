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
	personsDelivery := delivery.NewHttpDelivery(proxyServer, logger, personsUseCase)

	repeater := echo.New()
	repeater.POST("/repeat/:id", personsDelivery.SendRequest)
	repeater.GET("/secure/:id", personsDelivery.ParmMine)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		logger.Debug(proxyServer.Start(port))
		wg.Done()
	}()

	go func () {
		logger.Debug(repeater.Start(":8081"))
		wg.Done()
	}()

	wg.Wait()
}
