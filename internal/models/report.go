package models

import "time"

// EmployeeDepartment — справочник отделов (из поля department источника).
type EmployeeDepartment struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// Employee — сотрудник из сторонней БД (Users без password).
type Employee struct {
	ID             int64
	SourceID       int64 // Users.id на источнике
	Username       string
	Longname       string
	City           string
	Email          string
	IsAdmin        bool
	IsBlocked      bool
	DepartmentID   *int64 // FK на employee_departments
	DepartmentName string // для чтения (JOIN), не хранится
	SyncedAt       time.Time
}

// EmployeeWorkType — справочник WorkTypes источника.
type EmployeeWorkType struct {
	SourceID       int32 // WorkTypes.id
	Name           string
	CalculateHours bool
}

// EmployeeProject — справочник Projects источника.
type EmployeeProject struct {
	PrjID       string // Projects.prj_id (0 = «Не учтено»)
	Description string
	SyncedAt    time.Time
}

// EmployeeReport — сырая запись Reports источника.
type EmployeeReport struct {
	SourceID    int64  // Reports.id
	EmployeeID  int64  // FK на employees.id (локальный)
	PrjID       string // 0 = «Не учтено»
	WorkTypeID  int32  // FK на employee_work_types.source_id
	StartTime   time.Time
	EndTime     time.Time
	Description string
	SourceTS    *time.Time // Reports.timestamp
	SyncedAt    time.Time
	WorkKind    string // маппинг department → work_kind, сохраняется в БД
	StageNumber string // этап, указанный РП (может быть пустым)

	// Денормализованные поля для чтения (заполняются JOIN'ом, не хранятся).
	EmployeeName   string
	WorkTypeName   string
	CalculateHours bool
	DepartmentName string
}

type DepartmentWorkKind struct {
	DepartmentID int64
	WorkKind     string
}

type EmployeeRate struct {
	ID          int64
	EmployeeID  int64
	RatePerHour float64
	ValidFrom   time.Time
}

// WorklogRow — расчётная строка табеля (на лету, не хранится).
// Соответствует логике вьюхи MainExport источника.
type WorklogRow struct {
	SourceID       int64
	ReportDate     time.Time // дата StartTime (локальная)
	Weekday        string    // ПН..ВС
	StartTime      time.Time
	EndTime        time.Time
	Hours          float64 // с вычетом обеда, если calculate_hours
	Overwork       float64 // max(raw-9, 0), если calculate_hours
	Pause          int     // 1 если raw>=9 и calculate_hours, иначе 0
	Place          string  // WorkTypeName
	PrjID          string
	Description    string
	EmployeeName   string
	DepartmentName string
	WorkKind       string
	StageNumber    string
}
