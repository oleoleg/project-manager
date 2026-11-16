package models

import "time"

type Stage struct {
	GUID               string
	ContractID         string
	RowOrder           int
	StageNumber        string
	Name               string
	Cost               float64
	ClosedByActs       float64
	ClosedByActsManual bool
	VATPercent         float64
	AdvancePercent     float64
	StartConditions    string
	EndConditions      string
	CloseDatePlan      *time.Time
	ResponsiblePerson  string
	StatusCode         int16
	WorkKind           string
	Note               string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// StageStatusRef — справочник статусов этапа
type StageStatusRef struct {
	Code int16
	Name string
}

func GetStageStatuses() []StageStatusRef {
	return []StageStatusRef{
		{Code: 0, Name: "Не активен"},
		{Code: 1, Name: "В работе"},
		{Code: 2, Name: "Подписание актов"},
		{Code: 3, Name: "Закрыт"},
		{Code: 4, Name: "Закрыт, но есть долги"},
		{Code: 5, Name: "Гарантия"},
		{Code: 6, Name: "Аннулирован"},
	}
}

// WorkKind — справочник видов работ
type WorkKind struct {
	Code string
	Name string
}

// Список видов работ
var WorkKinds = []WorkKind{
	{Code: "Сборка", Name: "Сборка"},
	{Code: "Комплексный", Name: "Комплексный"},
	{Code: "ПИР", Name: "ПИР"},
	{Code: "ПНР", Name: "ПНР"},
	{Code: "ППО", Name: "ППО"},
	{Code: "Ремонт", Name: "Ремонт"},
	{Code: "Продажа", Name: "Продажа"},
	{Code: "СМР", Name: "СМР"},
	{Code: "ТО", Name: "ТО"},
}

// GetWorkKindName возвращает имя по коду (на случай, если код и имя отличаются)
func GetWorkKindName(code string) string {
	for _, wk := range WorkKinds {
		if wk.Code == code {
			return wk.Name
		}
	}
	return code
}
