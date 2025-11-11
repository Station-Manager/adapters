package common

import (
	"github.com/Station-Manager/adapters/converters"
	"github.com/Station-Manager/errors"
	"github.com/aarondl/null/v8"
)

// TypeToModelStringConverter converts a string to a model null.String.
func TypeToModelStringConverter(src any) (any, error) {
	const op errors.Op = "converters.common.TypeToModelCountryConverter"
	srcVal, ok := src.(string)
	if !ok {
		return null.String{}, errors.New(op).Errorf("Given parameter not a string, got %T", src)
	}
	// Treat empty string as null (same behavior) but do not error
	if srcVal == "" {
		return null.String{}, nil
	}
	return null.StringFrom(srcVal), nil
}

// ModelToTypeStringConverter converts a model null.String to a string.
func ModelToTypeStringConverter(src any) (any, error) {
	const op errors.Op = "converters.common.ModelToTypeCountryConverter"

	// Handle null.String type
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
