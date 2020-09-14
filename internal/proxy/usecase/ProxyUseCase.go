package usecase

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/pycnick/proxy/internal/proxy"
	"github.com/pycnick/proxy/internal/proxy/models"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
)

type ProxyUseCase struct {
	pR  proxy.Repository
	log *logrus.Logger
}

func NewProxyUseCase(log *logrus.Logger, pR proxy.Repository) *ProxyUseCase {
	return &ProxyUseCase{
		pR:  pR,
		log: log,
	}
}

func (pUC *ProxyUseCase) HandleRequest(httpRequest *models.HttpRequest) (*models.HttpResponse, error) {
	response, err := pUC.pR.SendHttpRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	err = pUC.pR.Create(httpRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (pUC *ProxyUseCase) HandleHttpsTunnel(httpRequest *models.HttpRequest, clientConn net.Conn) error {
	destConn, err := pUC.pR.CreateTcpConnection(httpRequest.Host)
	if err != nil {
		return err
	}

	if err := pUC.pR.Create(httpRequest); err != nil {
		return err
	}

	go func(destination io.WriteCloser, source io.ReadCloser) {
		defer destination.Close()
		defer source.Close()
		buf := new(bytes.Buffer)
		w := io.MultiWriter(destination, buf)
		io.Copy(w, source)
		r := &http.Request{}
		err := r.Write(buf)
		fmt.Println(r, err)
	}(destConn, clientConn)

	go func(destination io.WriteCloser, source io.ReadCloser) {
		defer destination.Close()
		defer source.Close()
		io.Copy(destination, source)
	}(clientConn, destConn)

	return nil
}

func (pUC *ProxyUseCase) GetHistory() ([]*models.HttpRequest, error) {
	requests, err := pUC.pR.ReadAll()
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (pUC *ProxyUseCase) RepeatRequest(ID uuid.UUID) (*models.HttpResponse, error) {
	request, err := pUC.pR.ReadByID(ID)
	if err != nil {
		return nil, err
	}

	response, err := pUC.pR.SendHttpRequest(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
