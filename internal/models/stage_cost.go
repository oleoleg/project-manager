package models

import "time"

type StageCost struct {
	GUID           string
	StageGUID      string
	WorkKindCode   string
	Cost           float64
	FOTPlan        float64
	FOTRatio       float64
	LaborDays      float64
	RatePerDay     float64
	TravelExpenses float64
	EquipmentSale  float64
	EquipmentCost  float64
	Subcontract    float64
	Commission     float64
	OtherExpenses  float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
