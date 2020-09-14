package proxy

import (
	"github.com/google/uuid"
	"github.com/pycnick/proxy/internal/proxy/models"
	"net"
)

type UseCase interface {
	HandleRequest(httpRequest *models.HttpRequest) (*models.HttpResponse, error)
	HandleHttpsTunnel(httpRequest *models.HttpRequest, clientConn net.Conn) error
	RepeatRequest(ID uuid.UUID) (*models.HttpResponse, error)
	GetHistory() ([]*models.HttpRequest, error)
}
