package utils

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var TimeTo = timeTo{}

type timeTo struct{}

func (t2 timeTo) PGTimestamptz(t time.Time) (pgtype.Timestamptz, error) {
	var tsz pgtype.Timestamptz
	if err := tsz.Scan(t); err != nil {
		return pgtype.Timestamptz{}, err
	}
	tsz.Valid = true
	return tsz, nil
}

func (t2 timeTo) PGTimestamp(t time.Time) (pgtype.Timestamp, error) {
	var ts pgtype.Timestamp
	if err := ts.Scan(t); err != nil {
		return pgtype.Timestamp{}, err
	}
	ts.Valid = true
	return ts, nil
}
