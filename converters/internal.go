package converters

import "github.com/Station-Manager/errors"

func checkString(src any) (string, error) {
	const op errors.Op = "adapters.checkString"
	srcVal, ok := src.(string)
	if !ok {
		return "", errors.New(op).Errorf("Given parameter not a string, got %T", src)
	}
	if srcVal == "" {
		return "", errors.New(op).Msg(errMsgFreqParamEmpty)
	}
	return srcVal, nil
}
