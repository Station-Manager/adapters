package sqlite

import (
	"github.com/Station-Manager/adapters/converters"
	"github.com/Station-Manager/errors"
	"time"
)

func TypeToModelDateConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.TypeToModelDateConverter"
	srcVal, err := converters.CheckString(src)
	if err != nil {
		return "", errors.New(op).Err(err)
	}

	// Accept multiple date formats and convert to YYYYMMDD
	var retVal time.Time
	switch len(srcVal) {
	case 8:
		// YYYYMMDD format
		retVal, err = time.Parse("20060102", srcVal)
	case 10:
		// Try YYYY-MM-DD format
		if srcVal[4] == '-' && srcVal[7] == '-' {
			retVal, err = time.Parse("2006-01-02", srcVal)
		} else {
			err = errors.New(op).Msg(converters.ErrMsgBadDateFormat)
		}
	default:
		return "", errors.New(op).Msg(converters.ErrMsgBadDateFormat)
	}

	if err != nil {
		return "", errors.New(op).Err(err).Msg(converters.ErrMsgBadDateFormat)
	}
	return retVal.Format("20060102"), nil
}

func TypeToModelTimeConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.TypeToModelTimeConverter"
	srsVal, err := converters.CheckString(src)
	if err != nil {
		return nil, errors.New(op).Err(err)
	}

	// Accept both HH:MM and HHMM formats
	var retVal time.Time
	if len(srsVal) == 5 && srsVal[2] == ':' {
		// HH:MM format - parse and convert to HHMM
		retVal, err = time.Parse("15:04", srsVal)
		if err != nil {
			return nil, errors.New(op).Err(err).Msg(converters.ErrMsgBadTimeFormat)
		}
	} else if len(srsVal) == 4 {
		// HHMM format
		retVal, err = time.Parse("1504", srsVal)
		if err != nil {
			return nil, errors.New(op).Err(err).Msg(converters.ErrMsgBadTimeFormat)
		}
	} else {
		return nil, errors.New(op).Msg(converters.ErrMsgBadTimeFormat)
	}

	return retVal.Format("1504"), nil
}

func ModelToTypeDateConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.ModelToTypeDateConverter"
	srcVal, err := converters.CheckString(src)
	if err != nil {
		return "", errors.New(op).Err(err)
	}

	if len(srcVal) != 8 {
		return "", errors.New(op).Msg(converters.ErrMsgBadDateFormat)
	}

	retVal, err := time.Parse("20060102", srcVal)
	if err != nil {
		return "", errors.New(op).Err(err).Msg(converters.ErrMsgBadDateFormat)
	}

	return retVal.Format("2006-01-02"), nil
}
