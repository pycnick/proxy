package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pycnick/proxy/internal/proxy/models"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
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
	request, err := http.NewRequest(httpRequest.Method, httpRequest.URL.Scheme + httpRequest.URL.Host + httpRequest.URL.Path, httpRequest.Body)
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

func (pR *ProxyRepository) CreateTcpConnection(host string) (net.Conn, error) {
	destConn, err := net.DialTimeout("tcp", host, time.Second*10)
	if err != nil {
		pR.log.Error(err)
		return nil, err
	}

	return destConn, nil
}
