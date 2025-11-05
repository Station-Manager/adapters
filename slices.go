package adapters

import (
	"fmt"
	models "github.com/7Q-Station-Manager/database/sqlite/models"
	"github.com/7Q-Station-Manager/types"
	"github.com/7Q-Station-Manager/utils"
	"html"
)

// QsoModelSliceToQsoTypeSlice converts a slice of Qso models to a slice of Qso types with proper transformation and validation.
// It handles string encoding, limits the length of specific fields, and formats date and time values.
// Returns the transformed slice or an error if any model cannot be converted properly.
func QsoModelSliceToQsoTypeSlice(slice models.QsoSlice) (types.QsoSlice, error) {
	var qsoList []types.Qso
	for _, model := range slice {
		item, err := ConvertModelToType[types.Qso](model)
		if err != nil {
			return nil, fmt.Errorf("error converting Qso model to Qso type: %w", err)
		}

		//		item.Name = html.EscapeString(item.Name)
		if item.Name, err = utils.DecodeStringToUTF8(item.Name); err != nil {
			item.Name = html.EscapeString(item.Name)
		}
		if len(item.Name) > 100 {
			item.Name = item.Name[:100]
		}

		item.QsoDate = utils.FormatDate(item.QsoDate)
		item.QsoDateOff = utils.FormatDate(item.QsoDateOff)
		item.TimeOn = utils.FormatTime(item.TimeOn)
		item.TimeOff = utils.FormatTime(item.TimeOff)

		qsoList = append(qsoList, *item)
	}

	return qsoList, nil
}
