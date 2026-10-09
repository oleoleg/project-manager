package models

import "time"

type User struct {
	ID           int64
	Username     string
	FullName     string
	PasswordHash string
	RoleCode     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Роли
const (
	RoleAdmin      = "admin"
	RoleRPCheief   = "rp_chief"
	RoleDirector   = "director"
	RoleRP         = "rp"
	RoleAccountant = "accountant"
)

// RoleName — человекочитаемое название роли.
func RoleName(code string) string {
	switch code {
	case RoleAdmin:
		return "Администратор"
	case RoleRPCheief:
		return "Начальник РП"
	case RoleDirector:
		return "Директор"
	case RoleRP:
		return "Руководитель проектов"
	case RoleAccountant:
		return "Бухгалтер"
	}
	return code
}
