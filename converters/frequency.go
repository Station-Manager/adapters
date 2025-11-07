package converters

import (
	"github.com/Station-Manager/errors"
	"strconv"
)

func TypeToModelFreqConverter(src any) (any, error) {
	const op errors.Op = "converters.TypeToModelFreqConverter"
	srcVal, err := checkString(src)
	if err != nil {
		return 0, errors.New(op).Err(err)
	}
	retVal, err := strconv.ParseFloat(srcVal, 64)
	if err != nil {
		return 0, errors.New(op).Err(err)
	}
	return retVal, nil
}
