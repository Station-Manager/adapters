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

func ModelToTypeFreqConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.ModelToTypeFreqConverter"
	//srcVal, err := converters.CheckFloat64(src)
	//if err != nil {
	//	return "", errors.New(op).Err(err)
	//}
	//	retVal := strconv.FormatInt(srcVal, 10)
	return "", nil
}
