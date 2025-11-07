package converters

import (
	"github.com/Station-Manager/errors"
	"time"
)

func TypeToModelDateConverter(src any) (any, error) {
	const op errors.Op = "converters.TypeToModelDateConverter"
	srcVal, err := checkString(src)
	if err != nil {
		return "", errors.New(op).Err(err)
	}
	retVal, err := time.Parse("20060102", srcVal)
	if err != nil {
		return "", errors.New(op).Err(err)
	}
	return retVal, nil
}

func TypeToModelTimeConverter(src any) (any, error) {
	const op errors.Op = "converters.TypeToModelTimeConverter"
	srsVal, err := checkString(src)
	if err != nil {
		return nil, errors.New(op).Err(err)
	}

	retVal, err := time.Parse("1504", srsVal)
	if err != nil {
		return nil, errors.New(op).Err(err)
	}

	result := time.Date(0, 0, 0, retVal.Hour(), retVal.Minute(), 0, 0, time.UTC)
	return result, nil
}
