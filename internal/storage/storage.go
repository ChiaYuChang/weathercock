package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/ChiaYuChang/weathercock/internal/models"
	ec "github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/jackc/pgerrcode"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	db    *models.Queries
	Cache *redis.Client
}

func New(db models.DBTX, cache *redis.Client) Storage {
	return Storage{
		db:    models.New(db),
		Cache: cache,
	}
}

func handlePgxErr(err error) *ec.Error {
	if err == nil {
		return nil
	}

	if pgerr, ok := ec.NewPGErr(err); ok {
		fmt.Println("convert error to PGErr:", pgerr)
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

	fmt.Println("Failed to convert error to PGErr, falling back to generic error handling")
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
