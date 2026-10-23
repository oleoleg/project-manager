package models

import (
	"time"
)

// BudgetSettings — настройки бюджета для конкретного договора.
type BudgetSettings struct {
	ContractID          string
	ProjectStartDate    time.Time
	ProjectEndDate      *time.Time
	PaymentDelayDays    int
	EquipPaymentPercent float64
	EquipPaymentDelay   int
	OverheadRate        float64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// BudgetPlanSnapshot — снимок (версия) сгенерированного ФинПлана.
type BudgetPlanSnapshot struct {
	ID           string
	ContractID   string
	CreatedAt    time.Time
	CreatedBy    *int64
	CreatorName  string
	Note         string
	IsActive     bool
	Settings     []byte
	ApprovedBy   *int64
	ApprovedAt   *time.Time
	ApproverName string
}

// BudgetPlanLine — строка сгенерированного ФинПлана.
type BudgetPlanLine struct {
	ID            string
	SnapshotID    string
	StageGUID     *string // <--- Исправили на string
	WorkKindCode  string
	Name          string
	OperationDate time.Time
	DelayDays     int
	PaymentDate   time.Time
	Income        float64
	Expense       float64
	Balance       float64
	OperationType string
	Note          string
}

// BudgetFactLine — фактическая операция.
type BudgetFactLine struct {
	ID            string
	ContractID    string
	StageGUID     *string // <--- Исправили на string
	WorkKindCode  string
	Name          string
	OperationDate time.Time
	DelayDays     int
	PaymentDate   time.Time
	Income        float64
	Expense       float64
	Balance       float64
	OperationType string
	Note          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
