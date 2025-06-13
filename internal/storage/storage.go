package storage

import (
	"database/sql"
	"errors"

	"github.com/ChiaYuChang/weathercock/internal/models"
	ec "github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	DB    *models.Queries
	Cache *redis.Client
}

func New(db models.DBTX, cache *redis.Client) Storage {
	return Storage{
		DB:    models.New(db),
		Cache: cache,
	}
}

func handlePgxErr(err error) *ec.Error {
	if pgerr, ok := ec.NewPGErr(err); ok {
		e := ec.ErrDBError.Clone().
			WithMessage(pgerr.Message).
			WithDetails(pgerr.Details).
			Warp(err)
		return e
	}

	if errors.Is(err, sql.ErrNoRows) {
		e := ec.ErrNotFound.Clone().
			Warp(err)
		return e
	}

	e := ec.ErrDBError.Clone().
		WithDetails(err.Error()).
		Warp(err)
	return e
}
