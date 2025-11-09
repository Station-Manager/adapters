package converters

import (
	"github.com/Station-Manager/errors"
)

func CheckString(op errors.Op, src any) (string, error) {
	srcVal, ok := src.(string)
	if !ok {
		return "", errors.New(op).Errorf("Given parameter not a string, got %T", src)
	}
	if srcVal == "" {
		return "", errors.New(op).Msg(ErrMsgFreqParamEmpty)
	}
	return srcVal, nil
}

func CheckFloat64(src any) (float64, error) {
	const op errors.Op = "converters.CheckFloat64"
	srcVal, ok := src.(float64)
	if !ok {
		return 0, errors.New(op).Errorf("Given parameter not a float64, got %T", src)
	}
	if srcVal == 0 {
		return 0, errors.New(op).Msg(ErrMsgFreqParamEmpty)
	}
	return srcVal, nil
}

func CheckInt64(op errors.Op, src any) (int64, error) {
	srcVal, ok := src.(int64)
	if !ok {
		return -1, errors.New(op).Errorf("Given parameter not a int64, got %T", src)
	}
	return srcVal, nil
}
