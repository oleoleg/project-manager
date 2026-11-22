package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

// BaseData собирает общие данные для шаблонов (заголовок, текущий пользователь, имя роли).
func BaseData(r *http.Request, title string) map[string]any {
	u := middleware.UserFromContext(r.Context())
	data := map[string]any{
		"Title": title,
		"User":  u,
		"RoleLabel": func(code string) string {
			return models.RoleName(code)
		},
	}
	if u != nil {
		data["RoleName"] = models.RoleName(u.RoleCode)
	}
	return data
}

// RenderTemplate рендерит шаблон и логирует ошибку, если она есть.
func RenderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

// ContractBaseData собирает общие данные для страниц, связанных с договором.
// Автоматически загружает контрагента по c.CounterpartyID.
func ContractBaseData(r *http.Request, title string, c *models.Contract, cpSvc *services.CounterpartyService) map[string]any {
	data := BaseData(r, title)

	statuses := models.GetContractStatuses()
	types := models.GetContractTypes()

	data["Contract"] = c
	data["ContractStatuses"] = statuses
	data["Types"] = types
	data["StatusName"] = findStatusName(statuses, c.StatusCode)
	data["TypeName"] = findTypeName(types, c.ContractType)

	if c != nil && c.CounterpartyID != "" {
		cp, _ := cpSvc.GetByID(r.Context(), c.CounterpartyID)
		data["Counterparty"] = cp
	} else {
		data["Counterparty"] = nil
	}

	return data
}

// CanEditContract возвращает true, если текущий пользователь может
// редактировать договор: admin, rp_chief или назначенный ответственный.
func CanEditContract(r *http.Request, c *models.Contract) bool {
	u := middleware.UserFromContext(r.Context())
	if u == nil || c == nil {
		return false
	}
	if u.RoleCode == "admin" || u.RoleCode == "rp_chief" || u.RoleCode == "director" {
		return true
	}
	if c.ResponsibleUserID != nil && *c.ResponsibleUserID == u.ID {
		return true
	}
	return false
}
