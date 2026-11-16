package models

import "time"

type Contract struct {
	ID                    string
	ContractNumber        string
	ContractDate          *time.Time // может быть пустой
	WorkName              string
	ResponsiblePerson     string
	ResponsibleUserID     *int64 // FK на users
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

// ContractStatusRef — справочник статусов договора
type ContractStatusRef struct {
	Code int16
	Name string
}

func GetContractStatuses() []ContractStatusRef {
	return []ContractStatusRef{
		{Code: 0, Name: "Заключение договора"},
		{Code: 1, Name: "В работе"},
		{Code: 2, Name: "Закрытие договора"},
		{Code: 3, Name: "Закрыт"},
		{Code: 4, Name: "Закрыт, но есть долги"},
		{Code: 5, Name: "Не пошло в реализацию"},
		{Code: 6, Name: "Расторжение"},
	}
}

// ContractTypeRef — справочник типов договоров
type ContractTypeRef struct {
	Code string
	Name string
}

func GetContractTypes() []ContractTypeRef {
	return []ContractTypeRef{
		{Code: "1", Name: "Подряд"},
		{Code: "2", Name: "Поставка"},
		{Code: "3", Name: "Сервис"},
		{Code: "4", Name: "Остальное"},
	}
}

// TrackingStatusRef — статусы (финплан, план-график)
type TrackingStatusRef struct {
	Code string
	Name string
}

func GetTrackingStatuses() []TrackingStatusRef {
	return []TrackingStatusRef{
		{Code: "none", Name: "Не отслеживать"},
		{Code: "active", Name: "Отслеживать"},
	}
}

// ProcurementCardStatusRef — статусы карты закупки
type ProcurementCardStatusRef struct {
	Code string
	Name string
}

func GetProcurementCardStatuses() []ProcurementCardStatusRef {
	return []ProcurementCardStatusRef{
		{Code: "active", Name: "Ведется"},
		{Code: "closed", Name: "Закрыта"},
		{Code: "none", Name: "Не ведется"},
	}
}
