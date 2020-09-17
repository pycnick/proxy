package usecase

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"github.com/google/uuid"
	"github.com/pycnick/proxy/internal/proxy"
	"github.com/pycnick/proxy/internal/proxy/models"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
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

func (pUC *ProxyUseCase) HandleRequest(httpRequest *http.Request) (*http.Response, error) {
	response, err := pUC.pR.SendHttpRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, httpRequest.Body); err != nil {
		return nil, err
	}
	err = pUC.pR.Create(&models.HttpRequest{
		Method:  httpRequest.Method,
		Schema:  httpRequest.URL.Scheme,
		Host:    httpRequest.Host,
		Path:    httpRequest.URL.Path,
		Headers: httpRequest.Header,
		Body:    buf.String(),
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (pUC *ProxyUseCase) handleHttpsRequest(clientReader *bufio.Reader, clientWriter *bufio.Writer, connectReq *http.Request) error{
	for {
		r, err := http.ReadRequest(clientReader)
		if err != nil {
			pUC.log.Error(err)
			return err
		}

		requestBody := new(bytes.Buffer)
		_, err = io.Copy(requestBody, r.Body)

		request := &models.HttpRequest{
			Method:  r.Method,
			Schema:  "https://",
			Host:    connectReq.Host,
			Path:    r.URL.Path,
			Headers: r.Header,
			Body:    requestBody.String(),
		}

		if err := pUC.pR.Create(request); err != nil {
			pUC.log.Error(err)
		}

		httpsReq := &http.Request{
			Method:           request.Method,
			URL:              &url.URL{
				Scheme:      request.Schema,
				Host:        request.Host,
				Path:        request.Path,
			},
			Header:           request.Headers,
			Body:             ioutil.NopCloser(strings.NewReader(request.Body)),
			Host:             request.Host,
		}

		response, err := pUC.pR.SendHttpRequest(httpsReq)
		if err != nil {
			pUC.log.Error(err)
			return err
		}

		if err := response.Write(clientWriter); err != nil {
			pUC.log.Error(err)
		}

		if err := clientWriter.Flush(); err != nil {
			pUC.log.Error(err)
		}
	}
}

func (pUC *ProxyUseCase) HandleHttpsConn(clientConn net.Conn, connectReq *http.Request) error {
	cert, err := tls.LoadX509KeyPair("./certs/ca.crt", "./certs/ca.key")
	if err != nil {
		pUC.log.Error(err)
	}

	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{cert},
		InsecureSkipVerify: true,
	})

	clientTlsReader := bufio.NewReader(tlsConn)
	clientTlsWriter := bufio.NewWriter(tlsConn)
	tlsConn.Handshake()

	c := make(chan bool)
	go func() {
		err := pUC.handleHttpsRequest(clientTlsReader, clientTlsWriter, connectReq)
		pUC.log.Error(err)
		c <- true
	}()
	<-c
	tlsConn.Close()
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

	response, err := pUC.pR.SendHttpRequest(&http.Request{
		Method:           request.Method,
		URL:              &url.URL{
			Scheme:      request.Schema,
			Host:        request.Host,
			Path:        request.Path,
		},
		Header:           request.Headers,
		Body:             ioutil.NopCloser(strings.NewReader(request.Body)),
		Host:             request.Host,
	})
	if err != nil {
		return nil, err
	}

	responseBody := new(bytes.Buffer)
	_, err = io.Copy(responseBody, response.Body)

	return &models.HttpResponse{
		Status:  response.StatusCode,
		Headers: response.Header,
		Body:    responseBody.String(),
	}, nil
}
