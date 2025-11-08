package sqlite

import (
	"github.com/Station-Manager/adapters/converters"
	"github.com/Station-Manager/errors"
	"github.com/aarondl/null/v8"
)

func TypeToModelCountryConverter(src any) (any, error) {
	const op errors.Op = "converters.sqlite.TypeToModelCountryConverter"
	srcVal, err := converters.CheckString(src)
	if err != nil {
		return null.String{}, errors.New(op).Err(err)
	}

	// If empty string, return null
	if srcVal == "" {
		return null.String{}, nil
	}

	// Return valid null.String
	return null.StringFrom(srcVal), nil
}
