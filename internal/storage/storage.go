package storage

import (
	"database/sql"
	"errors"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/ChiaYuChang/weathercock/internal/models"
	ec "github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	Queries *models.Queries
	Cache   *redis.Client
	db      *pgxpool.Conn
}

func New(conn *pgxpool.Conn, cache *redis.Client) Storage {
	return Storage{
		Queries: models.New(conn),
		Cache:   cache,
		db:      conn,
	}
}

func handlePgxErr(err error) *ec.Error {
	if err == nil {
		return nil
	}

	if pgerr, ok := ec.NewPGErr(err); ok {
		var e *ec.Error
		if pgerrcode.IsIntegrityConstraintViolation(pgerr.Code) {
			e = ec.ErrDBIntegrityConstrainViolation.Clone()
		} else {
			e = ec.ErrDBTypeConversionError.Clone()
		}
		e.WithMessage(pgerr.Message).
			WithDetails(pgerr.Details).
			Warp(err)
		return e
	}

	global.Logger.Warn().
		Err(err).
		Msg("failed to convert error to PGErr, falling back to generic error handling")
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
