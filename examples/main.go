package main

import (
	"fmt"
	"github.com/Station-Manager/adapters"
	sqlmodels "github.com/Station-Manager/database/sqlite/models"
	"github.com/Station-Manager/types"
)

func main() {
	qsoType := &types.Qso{
		QsoDetails: types.QsoDetails{
			Band:    "20m",
			Freq:    "14.320",
			Mode:    "SSB",
			QsoDate: "20251107",
			RstRcvd: "59",
			RstSent: "57",
			TimeOn:  "1200",
			TimeOff: "1205",
		},
		ContactedStation: types.ContactedStation{
			Call: "M0CMC",
		},
		LoggingStation: types.LoggingStation{
			StationCallsign: "7Q5MLV",
		},
	}

	model := &sqlmodels.Qso{}

	adapter := adapters.New()

	err := adapter.Adapt(qsoType, model)
	if err != nil {
		panic(err)
	}

	fmt.Println(model)

}
