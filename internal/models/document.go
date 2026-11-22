package models

import "time"

type ContractDocument struct {
	ID           int64
	ContractID   string
	DocType      string // main, addendum, appendix
	Number       string
	Title        string
	CurrentStage string
	CurrentForm  string
	HolderInfo   string
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Подгружаемые списки для UI
	Versions []DocumentVersion
	History  []DocumentHistory
}

type DocumentVersion struct {
	ID           int64
	DocumentID   int64
	VersionNum   int
	RelPath      string
	FileName     string
	FileExt      string
	FormType     string
	Comment      string
	UploadedBy   *int64
	UploaderName string
	CreatedAt    time.Time
}

type DocumentHistory struct {
	ID            int64
	DocumentID    int64
	VersionID     *int64
	VersionNum    int
	Stage         string
	FormType      string
	RecipientInfo string
	Note          string
	CreatedBy     *int64
	AuthorName    string
	CreatedAt     time.Time
}

type RefItem struct {
	Code string
	Name string
}

func GetDocTypes() []RefItem {
	return []RefItem{
		{Code: "main", Name: "Основной договор"},
		{Code: "addendum", Name: "Доп. соглашение (ДС)"},
		{Code: "appendix", Name: "Приложение / Спецификация"},
	}
}

func GetDocStages() []RefItem {
	return []RefItem{
		{Code: "draft", Name: "Подготовка проекта"},
		{Code: "internal_review", Name: "Внутреннее согласование"},
		{Code: "client_review", Name: "На рассмотрении у контрагента"},
		{Code: "rework", Name: "На доработке / Правки"},
		{Code: "signing", Name: "На подписании"},
		{Code: "signed", Name: "Подписан"},
	}
}

func GetDocForms() []RefItem {
	return []RefItem{
		{Code: "draft", Name: "Электронный проект (Word/PDF)"},
		{Code: "edo", Name: "ЭДО (Диадок / СБИС)"},
		{Code: "original", Name: "Бумажный оригинал"},
		{Code: "scan", Name: "Скан-копия"},
		{Code: "copy", Name: "Копия"},
	}
}
