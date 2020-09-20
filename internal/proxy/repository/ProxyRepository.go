package repository

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pycnick/proxy/internal/proxy/models"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"
)

type ProxyRepository struct {
	pool *pgxpool.Pool
	log  *logrus.Logger
}

func NewProxyRepository(log *logrus.Logger, pool *pgxpool.Pool) *ProxyRepository {
	return &ProxyRepository{
		pool: pool,
		log:  log,
	}
}

func (pR *ProxyRepository) Create(httpRequest *models.HttpRequest) error {
	sqlQuery := `INSERT INTO requests (method, schema, host, path, headers, body) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	if err := pR.pool.QueryRow(context.Background(), sqlQuery,
		httpRequest.Method,
		httpRequest.Schema,
		httpRequest.Host,
		httpRequest.Path,
		httpRequest.Headers,
		httpRequest.Body).Scan(
		&httpRequest.Id); err != nil {
		return err
	}

	return nil
}

func (pR *ProxyRepository) ReadAll() ([]*models.HttpRequest, error) {
	requests := []*models.HttpRequest{}
	sqlQuery := `SELECT * FROM requests`

	rows, err := pR.pool.Query(context.Background(), sqlQuery)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		request := &models.HttpRequest{}
		if err := rows.Scan(&request.Id,
			&request.Method,
			&request.Schema,
			&request.Host,
			&request.Path,
			&request.Headers,
			&request.Body); err != nil {
			return nil, err
		}

		requests = append(requests, request)
	}

	return requests, nil
}

func (pR *ProxyRepository) ReadByID(ID uuid.UUID) (*models.HttpRequest, error) {
	httpRequest := &models.HttpRequest{}
	sqlQuery := `SELECT * FROM requests WHERE id = $1`
	if err := pR.pool.QueryRow(context.Background(), sqlQuery,
		ID).Scan(
		&httpRequest.Id,
		&httpRequest.Method,
		&httpRequest.Schema,
		&httpRequest.Host,
		&httpRequest.Path,
		&httpRequest.Headers,
		&httpRequest.Body); err != nil {
		return nil, err
	}

	return httpRequest, nil
}

func (pR *ProxyRepository) SendHttpRequest(httpRequest *http.Request) (*http.Response, error) {
	request, err := http.NewRequest(httpRequest.Method, httpRequest.URL.String(), httpRequest.Body)
	if err != nil {
		return nil, err
	}

	request.Header = httpRequest.Header

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (pR *ProxyRepository) GetHttpsConnection(host string) (*tls.Conn, error) {
	file, err := os.Open("./certs/ca.key")
	if err != nil {
		pR.log.Error(err)
		return nil, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		pR.log.Error(err)
		return nil, err
	}

	privPem, _ := pem.Decode(b)
	if privPem.Type != "RSA PRIVATE KEY" {
		pR.log.Error("RSA private key is of the wrong type", privPem.Type)
	}

	priv, err := x509.ParsePKCS1PrivateKey(privPem.Bytes)
	if err != nil {
		pR.log.Error(err)
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{host},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
	cert, err := tls.X509KeyPair(caPEM.Bytes(), b)
	conf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		//InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", host, conf)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return conn, nil
}

func (pR *ProxyRepository) CreateTcpConnection(host string) (net.Conn, error) {
	destConn, err := net.DialTimeout("tcp", host, time.Second*10)
	if err != nil {
		pR.log.Error(err)
		return nil, err
	}

	return destConn, nil
}
