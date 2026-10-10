package models

import "time"

type Stage struct {
	GUID              string
	ContractID        string
	RowOrder          int
	StageNumber       string
	Name              string
	Cost              float64
	ClosedByActs      float64
	VATPercent        float64
	AdvancePercent    float64
	StartConditions   string
	EndConditions     string
	CloseDatePlan     *time.Time
	ResponsiblePerson string
	StatusCode        int16
	WorkKindCode      string
	Note              string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type StageStatusRef struct {
	Code int16
	Name string
}

type WorkKindRef struct {
	Code      string
	Name      string
	IsComplex bool
}

// ContractCard — маленькая модель для вкладки «Карточка договора».
type ContractCard struct {
	ContractID      string
	OverheadPercent float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
