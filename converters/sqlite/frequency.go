package sqlite

import (
	"github.com/Station-Manager/adapters/converters"
	"github.com/Station-Manager/errors"
	"strconv"
)

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
	return retVal, nil
}

func ModelToTypeFreqConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.ModelToTypeFreqConverter"
	srcVal, err := converters.CheckFloat64(src)
	if err != nil {
		return "", errors.New(op).Err(err)
	}
	retVal := strconv.FormatFloat(srcVal, 'f', -1, 64)
	return retVal, nil
}
