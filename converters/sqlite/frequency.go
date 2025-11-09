package sqlite

import (
	"github.com/Station-Manager/adapters/converters"
	"github.com/Station-Manager/errors"
	"math"
	"strconv"
)

// TypeToModelFreqConverter converts a string source value into a float64 representation of frequency.
// The source value is expected to be a string representation of a frequency in MHz.
// Returns the converted frequency or an error if the source is invalid or conversion fails.
func TypeToModelFreqConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.TypeToModelFreqConverter"
	srcVal, err := converters.CheckString(src)
	if err != nil {
		return 0, errors.New(op).Err(err)
	}
	retVal, err := strconv.ParseFloat(srcVal, 64)
	if err != nil {
		return 0, errors.New(op).Err(err)
	}
	hz := int64(math.Round(retVal * 1e6))
	return hz, nil
}

// ModelToTypeFreqConverter converts an int64 frequency in Hz to a string representing frequency in MHz with 3 decimal places.
// Returns the converted string and an error if the input is not valid.
func ModelToTypeFreqConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.ModelToTypeFreqConverter"
	srcVal, err := converters.CheckInt64(src)
	if err != nil {
		return "", errors.New(op).Err(err)
	}
	val := float64(srcVal) / 1e6
	retVal := strconv.FormatFloat(val, 'f', 3, 64)
	return retVal, nil
}
