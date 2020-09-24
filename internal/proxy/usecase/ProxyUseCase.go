package usecase

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/google/uuid"
	"github.com/pycnick/proxy/internal/proxy"
	"github.com/pycnick/proxy/internal/proxy/models"
	"github.com/sirupsen/logrus"
	"github.com/tjarratt/babble"
	"golang.org/x/crypto/acme/autocert"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type ProxyUseCase struct {
	pR          proxy.Repository
	log         *logrus.Logger
	certManager *autocert.Manager
	checkParams []string
}

func NewProxyUseCase(log *logrus.Logger, pR proxy.Repository) (*ProxyUseCase, error) {
	var paramSlice []string

	file, err := os.Open("./resources/params")
	defer file.Close()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		paramSlice = append(paramSlice, scanner.Text())
	}
	return &ProxyUseCase{
		pR:  pR,
		log: log,
		certManager: &autocert.Manager{
			Prompt: autocert.AcceptTOS,
		},
		checkParams: paramSlice,
	}, nil
}

func (pUC *ProxyUseCase) HandleRequest(httpRequest *http.Request) (*http.Response, error) {
	response, err := pUC.pR.SendHttpRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, httpRequest.Body); err != nil {
		pUC.log.Error(err)
	}
	err = pUC.pR.Create(&models.HttpRequest{
		Method:  httpRequest.Method,
		Schema:  httpRequest.URL.Scheme + "://",
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

func generateCert(host string) (tls.Certificate, error) {
	cmd := exec.Command("sh", "gen-cert.sh", host, strconv.Itoa(rand.Int()))
	cmd.Dir = os.Getenv("PWD") + "/certs"
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(out)
		return tls.Certificate{}, err
	}

	keyPath := os.Getenv("PWD") + "/certs/cert.key"
	certPath := os.Getenv("PWD") + "/certs/gen/" + host + ".crt"
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

func (pUC *ProxyUseCase) handleHttpsRequest(conn *tls.Conn, connectReq *http.Request) error {
	clientReader := bufio.NewReader(conn)
	r, err := http.ReadRequest(clientReader)
	if err != nil {
		pUC.log.Error(err)
		return err
	}

	requestBody := new(bytes.Buffer)
	_, err = io.Copy(requestBody, r.Body)

	request := &models.HttpRequest{
		Method:  r.Method,
		Schema:  "https",
		Host:    connectReq.Host,
		Path:    r.URL.Path,
		Headers: r.Header,
		Body:    requestBody.String(),
	}

	if err := pUC.pR.Create(request); err != nil {
		pUC.log.Error(err)
	}

	httpsRequest := &http.Request{
		Method: request.Method,
		URL: &url.URL{
			Scheme: request.Schema,
			Host:   request.Host,
			Path:   request.Path,
		},
		Header: request.Headers,
		Body:   ioutil.NopCloser(strings.NewReader(request.Body)),
		Host:   request.Host,
	}

	destConn, err := pUC.pR.GetHttpsConnection(httpsRequest.Host)
	if err != nil {
		pUC.log.Error(err)
		return err
	}

	if err := httpsRequest.Write(destConn); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer destConn.Close()
		defer conn.Close()
		io.Copy(destConn, conn)
		wg.Done()
	}()

	go func() {
		defer destConn.Close()
		defer conn.Close()
		io.Copy(conn, destConn)
		wg.Done()
	}()

	wg.Wait()

	return nil
}

func (pUC *ProxyUseCase) HandleHttpsConn(clientConn net.Conn, connectReq *http.Request) error {
	cert, err := generateCert(connectReq.Host[:strings.IndexByte(connectReq.Host, ':')])
	if err != nil {
		pUC.log.Error(err)
		return err
	}

	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{cert},
	})
	err = tlsConn.Handshake()

	c := make(chan bool)
	go func() {
		err := pUC.handleHttpsRequest(tlsConn, connectReq)
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
		Method: request.Method,
		URL: &url.URL{
			Scheme: request.Schema,
			Host:   request.Host,
			Path:   request.Path,
		},
		Header: request.Headers,
		Body:   ioutil.NopCloser(strings.NewReader(request.Body)),
		Host:   request.Host,
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

func (pUC *ProxyUseCase) ParamsSecurityCheck(ID uuid.UUID) (map[string]string, error) {
	request, err := pUC.pR.ReadByID(ID)
	if err != nil {
		return nil, err
	}

	secureParams := make(map[string]string)

	b := babble.NewBabbler()
	b.Count = 1

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	for ind, currWord := range pUC.checkParams {
		pUC.log.Error(ind)
		wg.Add(1)
		word := currWord
		go func() {
			randParamValue := b.Babble()
			response, err := pUC.pR.SendHttpRequest(&http.Request{
				Method: request.Method,
				URL: &url.URL{
					Scheme: request.Schema,
					Host:   request.Host,
					Path:   request.Path + "?" + word + "=" + randParamValue,
				},
				Header: request.Headers,
				Body:   ioutil.NopCloser(strings.NewReader(request.Body)),
				Host:   request.Host,
			})

			if err != nil {
				pUC.log.Error(err)
				return
			}

			buf := new(bytes.Buffer)
			if err := response.Write(buf); err != nil {
				pUC.log.Error(err)
			}

			if strings.Contains(buf.String(), randParamValue) {
				mu.Lock()
				secureParams[word] = randParamValue
				mu.Unlock()
			}
			wg.Done()
		}()
	}

	wg.Wait()

	return secureParams, nil
}
