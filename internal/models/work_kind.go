package models

// WorkKind — справочник видов работ
type WorkKind struct {
	Code string
	Name string
}

// Список видов работ (захардкожен, вместо таблицы work_kinds)
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
