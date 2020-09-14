package proxy

import (
	"github.com/pycnick/proxy/internal/proxy/models"
	"github.com/google/uuid"
	"net"
)

type Repository interface {
	Create(httpRequest *models.HttpRequest) error
	ReadByID(ID uuid.UUID) (*models.HttpRequest, error)
	ReadAll() ([]*models.HttpRequest, error)

	CreateTcpConnection(host string) (net.Conn, error)
	SendHttpRequest(httpRequest *models.HttpRequest) (*models.HttpResponse, error)
}