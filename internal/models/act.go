package models

import "time"

// Виды актов
const (
	ActKindAcceptance   = "acceptance"    // Акт приема/сдачи
	ActKindUPD          = "upd"           // УПД
	ActKindInvoiceFact  = "invoice_fact"  // Счёт-фактура
	ActKindKS2          = "ks2"           // КС-2
	ActKindKS3          = "ks3"           // КС-3
	ActKindInvoice      = "invoice"       // Счёт
	ActKindPaymentOrder = "payment_order" // Платёжное поручение
	ActKindCustom       = "custom"        // Свой тип
)

// ActKindName — человекочитаемое имя вида акта.
func ActKindName(code string) string {
	switch code {
	case ActKindAcceptance:
		return "Акт приёма/сдачи"
	case ActKindUPD:
		return "УПД"
	case ActKindInvoiceFact:
		return "Счёт-фактура"
	case ActKindKS2:
		return "КС-2"
	case ActKindKS3:
		return "КС-3"
	case ActKindInvoice:
		return "Счёт"
	case ActKindPaymentOrder:
		return "Платёжное поручение"
	case ActKindCustom:
		return "Свой тип"
	}
	return code
}

// ActKindRef — справочник для select'ов.
type ActKindRef struct {
	Code string
	Name string
}

// ActKindRefs — все виды актов.
func ActKindRefs() []ActKindRef {
	return []ActKindRef{
		{ActKindAcceptance, ActKindName(ActKindAcceptance)},
		{ActKindUPD, ActKindName(ActKindUPD)},
		{ActKindInvoiceFact, ActKindName(ActKindInvoiceFact)},
		{ActKindKS2, ActKindName(ActKindKS2)},
		{ActKindKS3, ActKindName(ActKindKS3)},
		{ActKindInvoice, ActKindName(ActKindInvoice)},
		{ActKindPaymentOrder, ActKindName(ActKindPaymentOrder)},
		{ActKindCustom, ActKindName(ActKindCustom)},
	}
}

// Act — главный (денежный) акт.
type Act struct {
	GUID                 string
	Kind                 string
	KindCustom           string
	Name                 string
	Amount               float64 // без НДС
	SignDate             *time.Time
	FilePath             string
	Note                 string
	DistributionExpanded bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ActStageLink — распределение суммы акта по этапу.
type ActStageLink struct {
	ActGUID   string
	StageGUID string
	Amount    float64 // без НДС
}

// ActAttachment — приложение (информационный акт), привязано к главному.
type ActAttachment struct {
	GUID       string
	ActGUID    string
	Kind       string
	KindCustom string
	Name       string
	FilePath   string
	Note       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
