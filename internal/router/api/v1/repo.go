package api

import (
	"github.com/ChiaYuChang/weathercock/internal/storage"
	"github.com/nats-io/nats.go"
)

const (
	Version = "v1"
)

// Repo provides methods to interact with the database, cache and nats.
type Repo struct {
	Storage storage.Storage
	NATS    *nats.Conn
}

// NewRepo creates a new instance of Repo with the provided database and cache clients.
func NewRepo(s storage.Storage, nats *nats.Conn) *Repo {
	return &Repo{s, nats}
}

// Convert Repo to Tasks interface which provides methods for task management.
func (r *Repo) Task() Tasks {
	return Tasks(r)
}
