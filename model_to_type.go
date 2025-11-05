package adapters

import (
	"fmt"
	models "github.com/7Q-Station-Manager/database/sqlite/models"
	"github.com/7Q-Station-Manager/types"
)

// ConvertQSOModelSliceToQSOTypeSlice converts a slice of QSO models to a slice of QSO types, normalizing data during conversion.
// It returns the converted slice and an error if any part of the conversion fails.
func ConvertQSOModelSliceToQSOTypeSlice(slice models.QsoSlice) (types.QsoSlice, error) {
	var result types.QsoSlice

	for _, model := range slice {
		typeQso, err := ConvertModelToType[types.Qso](model)
		if err != nil {
			return nil, fmt.Errorf("converting qso model to qso type: %w", err)
		}
		qso := normalizeTypeQso(typeQso)
		result = append(result, qso)
	}

	return result, nil
}
