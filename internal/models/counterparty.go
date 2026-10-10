package models

import "time"

type Counterparty struct {
	ID             string
	Name           string
	Partner        string
	ShortName      string
	AddressActual  string
	AddressLegal   string
	URLWorkplace   string
	URLServicework string
	URLSalesprep   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
