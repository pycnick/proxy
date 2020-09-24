package proxy

import (
	"github.com/google/uuid"
	"github.com/pycnick/proxy/internal/proxy/models"
	"net"
	"net/http"
)

type UseCase interface {
	HandleRequest(httpRequest *http.Request) (*http.Response, error)
	HandleHttpsConn(clientConn net.Conn, connectReq *http.Request) error
	RepeatRequest(ID uuid.UUID) (*models.HttpResponse, error)
	GetHistory() ([]*models.HttpRequest, error)
	ParamsSecurityCheck(ID uuid.UUID) (map[string]string, error)
}
