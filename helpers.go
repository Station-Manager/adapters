package adapters

import (
	"github.com/7Q-Station-Manager/types"
	"github.com/7Q-Station-Manager/utils"
	"html"
)

func normalizeTypeQso(qso *types.Qso) types.Qso {
	var err error
	if qso.Name, err = utils.DecodeStringToUTF8(qso.Name); err != nil {
		// We are not interested in the error, just return EscapeString
		qso.Name = html.EscapeString(qso.Name)
	}
	if len(qso.Name) > 100 {
		qso.Name = qso.Name[:100]
	}
	qso.QsoDate = utils.FormatDate(qso.QsoDate)
	qso.QsoDateOff = utils.FormatDate(qso.QsoDateOff)
	qso.TimeOn = utils.FormatTime(qso.TimeOn)
	qso.TimeOff = utils.FormatTime(qso.TimeOff)
	return *qso // Return a copy
}
