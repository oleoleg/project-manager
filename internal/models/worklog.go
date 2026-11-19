package models

import "time"

// WorklogRow — расчётная строка табеля (на лету, не хранится).
// Соответствует логике вьюхи MainExport источника.
type WorklogRow struct {
	ReportDate     time.Time // дата StartTime (локальная)
	Weekday        string    // ПН..ВС
	StartTime      time.Time
	EndTime        time.Time
	Hours          float64 // с вычетом обеда, если calculate_hours
	Overwork       float64 // max(raw-9, 0), если calculate_hours
	Pause          int     // 1 если raw>=9 и calculate_hours, иначе 0
	Place          string  // WorkTypeName
	PrjID          int64
	Description    string
	EmployeeName   string
	DepartmentName string
	WorkKind       string
	StageNumber    string
}
