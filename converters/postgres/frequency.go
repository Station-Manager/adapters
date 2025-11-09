package postgres

import (
	"github.com/Station-Manager/adapters/converters"
	"github.com/Station-Manager/errors"
	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/ericlagergren/decimal"
	"strconv"
)

// TypeToModelFreqConverter converts a frequency value from a string to a decimal.
// It returns types.Decimal the underlying value is a decimal.Big and is retrieved as a float64.
// Both sqlite3 and postgres have the same type for frequencies, which are set by SQLBoiler.
func TypeToModelFreqConverter(src any) (any, error) {
	const op errors.Op = "converters.postgres.TypeToModelFreqConverter"
	srcVal, err := converters.CheckString(op, src)
	if err != nil {
		return 0, errors.New(op).Err(err)
	}
	retVal, err := strconv.ParseFloat(srcVal, 64)
	if err != nil {
		return 0, errors.New(op).Err(err)
	}
	val := types.NewDecimal(new(decimal.Big))
	val.SetFloat64(retVal)
	return val, nil
}
