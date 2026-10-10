package models

import "time"

type Contract struct {
	ID                    string
	ContractNumber        string
	ContractDate          *time.Time // может быть пустой
	WorkName              string
	ResponsiblePerson     string
	StatusCode            int16
	CounterpartyID        string
	Workshop              string
	ProjectURL            string
	ProcurementCardStatus *string
	ExtraDetails          string
	FinplanStatus         *string
	PlanScheduleStatus    *string
	ShortName             string
	ContractType          string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Справочные типы для select'ов
type ContractStatusRef struct {
	Code int16
	Name string
}

type ContractTypeRef struct {
	Code string
	Name string
}

type TrackingStatusRef struct {
	Code string
	Name string
}
