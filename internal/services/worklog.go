package services

import (
	"time"

	"github.com/oleoleg/project-manager/internal/models"
)

// BuildWorklogRow рассчитывает строку табеля по сырой записи.
// Логика соответствует вьюхе MainExport источника.
func BuildWorklogRow(r models.EmployeeReport) models.WorklogRow {
	raw := r.EndTime.Sub(r.StartTime).Hours() // float64

	var hours, overwork float64
	var pause int
	if r.CalculateHours {
		if raw > 8 {
			hours = raw - 1
		} else {
			hours = raw
		}
		if raw-9 > 0 {
			overwork = raw - 9
		}
		if raw >= 9 {
			pause = 1
		}
	}

	return models.WorklogRow{
		ReportDate:     r.StartTime,
		Weekday:        weekdayRU(r.StartTime.Weekday()),
		StartTime:      r.StartTime,
		EndTime:        r.EndTime,
		Hours:          hours,
		Overwork:       overwork,
		Pause:          pause,
		Place:          r.WorkTypeName,
		PrjID:          r.PrjID,
		Description:    r.Description,
		EmployeeName:   r.EmployeeName,
		DepartmentName: r.DepartmentName,
		WorkKind:       r.WorkKind,
		StageNumber:    r.StageNumber,
	}
}

func weekdayRU(d time.Weekday) string {
	switch d {
	case time.Monday:
		return "ПН"
	case time.Tuesday:
		return "ВТ"
	case time.Wednesday:
		return "СР"
	case time.Thursday:
		return "ЧТ"
	case time.Friday:
		return "ПТ"
	case time.Saturday:
		return "СБ"
	default:
		return "ВС"
	}
}
