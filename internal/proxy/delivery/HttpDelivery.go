package delivery

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pycnick/proxy/internal/proxy"
	"github.com/pycnick/proxy/internal/proxy/models"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
)

type HttpDelivery struct {
	pUC proxy.UseCase
	log *logrus.Logger
}

func NewHttpDelivery(e *echo.Echo, log *logrus.Logger, pUC proxy.UseCase) *HttpDelivery {
	hD := &HttpDelivery{
		pUC: pUC,
		log: log,
	}

	e.GET("/", hD.Proxy)
	e.POST("/", hD.Proxy)
	e.PUT("/", hD.Proxy)
	e.PATCH("/", hD.Proxy)
	e.HEAD("/", hD.Proxy)
	e.OPTIONS("/", hD.Proxy)
	e.DELETE("/", hD.Proxy)
	e.TRACE("/", hD.Proxy)

	e.CONNECT("", hD.ProxyTunnel)

	e.GET("/requests", hD.GetAllRequestsHistory)
	e.POST("/requests/:id", hD.SendRequest)
	return hD
}

func (hD *HttpDelivery) Proxy(c echo.Context) error {
	requestBody, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		logrus.Debug(err)
	}

	request := &models.HttpRequest{
		Method:  c.Request().Method,
		Schema:  "http://",
		Host:    c.Request().URL.Host,
		Path:    c.Request().URL.Path,
		Headers: c.Request().Header,
		Body:    string(requestBody),
	}

	response, err := hD.pUC.HandleRequest(request)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	for key, value := range response.Headers {
		c.Response().Header().Set(key, strings.Join(value, ","))
	}

	return c.String(response.Status, response.Body)
}

func (hD *HttpDelivery) ProxyTunnel(c echo.Context) error {
	c.String(http.StatusOK, "")

	requestBody, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		logrus.Debug(err)
	}

	request := &models.HttpRequest{
		Method:  c.Request().Method,
		Schema:  "https://",
		Host:    c.Request().URL.Host,
		Path:    c.Request().URL.Path,
		Headers: c.Request().Header,
		Body:    string(requestBody),
	}

	clientConn, _, err := c.Response().Hijack()
	if err != nil {
		hD.log.Error(err)
		return err
	}

	if err := hD.pUC.HandleHttpsTunnel(request, clientConn); err != nil {
		hD.log.Error(err)
		return err
	}

	return nil
}

func (hD *HttpDelivery) GetAllRequestsHistory(c echo.Context) error {
	requests, err := hD.pUC.GetHistory()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, requests)
}

func (hD *HttpDelivery) SendRequest(c echo.Context) error {
	requestID := c.Param("id")
	if requestID == "" {
		return c.String(http.StatusBadRequest, "")
	}

	requestUUID, err := uuid.Parse(requestID)
	if err != nil {
		return c.String(http.StatusBadRequest, "")
	}

	response, err := hD.pUC.RepeatRequest(requestUUID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	for key, value := range response.Headers {
		c.Response().Header().Set(key, strings.Join(value, ","))
	}

	return c.String(response.Status, response.Body)
}
