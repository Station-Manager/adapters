package adapters

import (
	"fmt"
	"github.com/Station-Manager/adapters/converters"
	sqlmodels "github.com/Station-Manager/database/sqlite/models"
	"github.com/Station-Manager/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"strconv"
	"testing"
	"time"
)

type TestSuite struct {
	suite.Suite
}

func TestRealworld(T *testing.T) {
	suite.Run(T, new(TestSuite))
}

//func (suite *TestSuite) TypeQsoToModelQsoNoAdditionalData() {
//	qsoType := &types.Qso{
//		QsoDetails: types.QsoDetails{
//			Band:    "20m",
//			Freq:    "14.320",
//			Mode:    "SSB",
//			QsoDate: "20251107",
//			RstRcvd: "59",
//			RstSent: "57",
//			TimeOn:  "1200",
//			TimeOff: "1205",
//		},
//		ContactedStation: types.ContactedStation{
//			Call: "M0CMC",
//		},
//		LoggingStation: types.LoggingStation{
//			StationCallsign: "7Q5MLV",
//		},
//	}
//
//	model := &sqlmodels.Qso{}
//
//	adapter := New()
//
//	err := adapter.Adapt(qsoType, model)
//	require.NoError(suite.T(), err)
//	require.Equal(suite.T(), qsoType.Band, model.Band)
//	require.Equal(suite.T(), qsoType.Freq, model.Freq)
//	require.Equal(suite.T(), qsoType.Mode, model.Mode)
//	require.Equal(suite.T(), qsoType.QsoDate, model.QsoDate)
//	require.Equal(suite.T(), qsoType.RstRcvd, model.RstRcvd)
//	require.Equal(suite.T(), qsoType.RstSent, model.RstSent)
//	require.Equal(suite.T(), qsoType.TimeOn, model.TimeOn)
//	require.Equal(suite.T(), qsoType.TimeOff, model.TimeOff)
//
//}

func (suite *TestSuite) TestTypeQsoToModelQsoWithAdditionalData() {
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
			Call:    "M0CMC",
			Cont:    "EU",
			Country: "England",
			Name:    "Marc",
		},
		LoggingStation: types.LoggingStation{
			MyAltitude:      "1311",
			MyAntenna:       "Hex Beam",
			MyCity:          "Mzuzu",
			MyCountry:       "Malawi",
			MyName:          "Marc",
			StationCallsign: "7Q5MLV",
		},
	}

	model := &sqlmodels.Qso{}

	adapter := New()
	adapter.RegisterConverter("Freq", converters.TypeToModelFreqConverter)
	adapter.RegisterConverter("QsoDate", converters.TypeToModelDateConverter)

	modelDate, err := time.Parse("20060102", qsoType.QsoDate)
	require.NoError(suite.T(), err)

	err = adapter.Adapt(qsoType, model)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), qsoType.Band, model.Band)
	require.Equal(suite.T(), 14.320, model.Freq)
	require.Equal(suite.T(), qsoType.Mode, model.Mode)
	require.Equal(suite.T(), modelDate, model.QsoDate)
	require.Equal(suite.T(), qsoType.RstRcvd, model.RstRcvd)
	require.Equal(suite.T(), qsoType.RstSent, model.RstSent)
	require.Equal(suite.T(), qsoType.TimeOn, model.TimeOn)
	require.Equal(suite.T(), qsoType.TimeOff, model.TimeOff)
}

func typeToModelFreqConverter(src any) (any, error) {
	srcVal, ok := src.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", src)
	}
	freq, err := strconv.ParseFloat(srcVal, 64)
	if err != nil {
		return nil, err
	}
	return freq, nil
}
