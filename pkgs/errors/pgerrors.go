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

type BatchErr struct {
	Errors map[int]error `json:"errors"`
}

func NewBatchErr() *BatchErr {
	return &BatchErr{
		Errors: make(map[int]error),
	}
}

func (b *BatchErr) Add(index int, err error) {
	if err == nil {
		return
	}
	if _, exists := b.Errors[index]; !exists {
		b.Errors[index] = err
	}
}

func (b *BatchErr) Error() string {
	if len(b.Errors) == 0 {
		return "no errors"
	}

	msg := "Batch errors:\n"
	for index, err := range b.Errors {
		msg += fmt.Sprintf("  - [%d] %s\n", index, err.Error())
	}
	return msg
}

func (b *BatchErr) IsEmpty() bool {
	return len(b.Errors) == 0
}

func (b *BatchErr) ToError() error {
	if b.IsEmpty() {
		return nil
	}

	e := ErrDBError.Clone()
	for index, err := range b.Errors {
		if pgErr, ok := NewPGErr(err); ok {
			e.WithDetails(fmt.Sprintf("Index %d: %s", index, pgErr.String()))
		} else {
			e.WithDetails(fmt.Sprintf("Index %d: %s", index, err.Error()))
		}
	}
	return e
}
