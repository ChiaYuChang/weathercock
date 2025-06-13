package errors

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

type PGErr struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Details  string `json:"details"`
}

func (p PGErr) String() string {
	return fmt.Sprintf("[%s][%s] %s, details: %s", p.Code, p.Severity, p.Message, p.Details)
}

func (p PGErr) Error() string {
	return p.String()
}

func NewPGErr(err error) (*PGErr, bool) {
	if err == nil {
		return nil, true
	}

	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); !ok {
		return nil, false
	}

	return &PGErr{
		Code:     pgErr.Code,
		Message:  pgErr.Message,
		Severity: pgErr.Severity,
		Details:  pgErr.Detail,
	}, true
}

func MustPGErr(err error) *PGErr {
	if err == nil {
		return nil
	}

	pgErr, ok := NewPGErr(err)
	if !ok {
		panic(fmt.Sprintf("error is not pg error: %v", err))
	}
	return pgErr
}
