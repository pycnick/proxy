package delivery

import (
	"bytes"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pycnick/proxy/internal/proxy"
	"github.com/sirupsen/logrus"
	"io"
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
	return hD
}

func (hD *HttpDelivery) Proxy(c echo.Context) error {
	response, err := hD.pUC.HandleRequest(c.Request())
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	for key, value := range response.Header {
		c.Response().Header().Set(key, strings.Join(value, ","))
	}

	buf := new(bytes.Buffer)
	io.Copy(buf, response.Body)
	return c.String(response.StatusCode, buf.String())
}

func (hD *HttpDelivery) ProxyTunnel(c echo.Context) error {
	c.String(http.StatusOK, "")

	clientConn, _, err := c.Response().Hijack()
	if err != nil {
		hD.log.Error(err)
		return err
	}

	if err := hD.pUC.HandleHttpsConn(clientConn, c.Request()); err != nil {
		hD.log.Error(err)
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

//func (hD *HttpDelivery) ParmMine(c echo.Context) error {
//	requestID := c.Param("id")
//	if requestID == "" {
//		return c.String(http.StatusBadRequest, "")
//	}
//
//	requestUUID, err := uuid.Parse(requestID)
//	if err != nil {
//		return c.String(http.StatusBadRequest, "")
//	}
//
//
//
//}
