package adapters

import (
	"github.com/Station-Manager/adapters/converters"
	sqlmodels "github.com/Station-Manager/database/sqlite/models"
	mytypes "github.com/Station-Manager/types"
	"github.com/aarondl/sqlboiler/v4/types"
	"github.com/ericlagergren/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	qsoType := &mytypes.Qso{
		QsoDetails: mytypes.QsoDetails{
			Band:    "20m",
			Freq:    "14.320",
			Mode:    "SSB",
			QsoDate: "20251107",
			RstRcvd: "59",
			RstSent: "57",
			TimeOn:  "1200",
			TimeOff: "1205",
		},
		ContactedStation: mytypes.ContactedStation{
			Call:    "M0CMC",
			Cont:    "EU",
			Country: "England",
			Name:    "Marc",
		},
		LoggingStation: mytypes.LoggingStation{
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
	adapter.RegisterConverter("TimeOn", converters.TypeToModelTimeConverter)
	adapter.RegisterConverter("TimeOff", converters.TypeToModelTimeConverter)

	freq := types.NewDecimal(new(decimal.Big))
	freq.SetFloat64(14.320)

	err := adapter.Adapt(qsoType, model)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), qsoType.Band, model.Band)

	// types.Decimal
	tVal, _ := freq.Float64()
	mVal, _ := model.Freq.Float64()
	require.Equal(suite.T(), tVal, mVal)

	// time.Time
	typeDate, err := time.Parse("20060102", qsoType.QsoDate)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), typeDate, model.QsoDate)

	// strings
	require.Equal(suite.T(), qsoType.Mode, model.Mode)
	require.Equal(suite.T(), qsoType.RstRcvd, model.RstRcvd)
	require.Equal(suite.T(), qsoType.RstSent, model.RstSent)

	// time.Time
	typeOnTime, err := time.Parse("1504", qsoType.TimeOn)
	require.NoError(suite.T(), err)
	typeOffTime, err := time.Parse("1504", qsoType.TimeOff)
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), typeOnTime, model.TimeOn)
	require.Equal(suite.T(), typeOffTime, model.TimeOff)
}
