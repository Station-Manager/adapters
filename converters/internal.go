package converters

import (
	"github.com/Station-Manager/errors"
)

func CheckString(src any) (string, error) {
	const op errors.Op = "converters.checkString"
	srcVal, ok := src.(string)
	if !ok {
		return "", errors.New(op).Errorf("Given parameter not a string, got %T", src)
	}
	if srcVal == "" {
		return "", errors.New(op).Msg(ErrMsgFreqParamEmpty)
	}
	return srcVal, nil
}
