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
	ID                 string
	ContractID         string
	CreatedAt          time.Time
	CreatedBy          *int64
	CreatorName        string
	Note               string
	IsActive           bool
	Settings           []byte
	StageCostsSnapshot []byte
	ApprovedBy         *int64
	ApprovedAt         *time.Time
	ApproverName       string
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
	ID            string    `json:"id"`
	ContractID    string    `json:"contract_id"`
	PlanLineID    *string   `json:"plan_line_id"`
	StageGUID     *string   `json:"stage_guid"`
	WorkKindCode  string    `json:"work_kind_code"`
	Name          string    `json:"name"`
	OperationDate time.Time `json:"operation_date"`
	DelayDays     int       `json:"delay_days"`
	PaymentDate   time.Time `json:"payment_date"`
	OperationType string    `json:"operation_type"`
	PlanIncome    float64   `json:"plan_income"`
	PlanExpense   float64   `json:"plan_expense"`
	FactIncome    float64   `json:"fact_income"`
	FactExpense   float64   `json:"fact_expense"`
	IsConfirmed   bool      `json:"is_confirmed"`
	IsManualEdit  bool      `json:"is_manual_edit"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
