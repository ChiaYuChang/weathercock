package api

import (
	"net/http"

	"github.com/ChiaYuChang/weathercock/internal/models"
	"github.com/google/uuid"
)

type TaskEndpoint interface {
	InsertFromText(r *http.Request) (uuid.UUID, error)
	InsertFromURL(r *http.Request) (uuid.UUID, error)
	Get(r *http.Request) (*models.UsersTask, error)
}

type UserArticlesEndpoint interface {
	GetByTaskID(r *http.Request) (*models.UsersArticle, error)
}
