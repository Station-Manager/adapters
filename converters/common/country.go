package common

import (
	"github.com/Station-Manager/adapters/converters"
	"github.com/Station-Manager/errors"
	"github.com/aarondl/null/v8"
)

func TypeToModelCountryConverter(src any) (any, error) {
	const op errors.Op = "converters.common.TypeToModelCountryConverter"
	srcVal, err := converters.CheckString(op, src)
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

func ModelToTypeCountryConverter(src any) (any, error) {
	const op errors.Op = "converters.common.ModelToTypeCountryConverter"

	// Handle null.String type from SQLite model
	if nullStr, ok := src.(null.String); ok {
		if !nullStr.Valid {
			return "", nil
		}
		return nullStr.String, nil
	}

	// Fallback to string check
	srcVal, err := converters.CheckString(op, src)
	if err != nil {
		return "", errors.New(op).Err(err)
	}

	return srcVal, nil
}
