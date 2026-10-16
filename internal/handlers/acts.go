package handlers

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type ActHandler struct {
	svc         *services.ActService
	stageSvc    *services.StageService
	contractSvc *services.ContractService
	tmpl        *template.Template
}

func NewActHandler(
	svc *services.ActService,
	stageSvc *services.StageService,
	contractSvc *services.ContractService,
	tmpl *template.Template,
) *ActHandler {
	return &ActHandler{svc: svc, stageSvc: stageSvc, contractSvc: contractSvc, tmpl: tmpl}
}

func (h *ActHandler) commonData(r *http.Request, title string) map[string]any {
	u := middleware.UserFromContext(r.Context())
	data := map[string]any{"Title": title, "User": u}
	if u != nil {
		data["RoleName"] = models.RoleName(u.RoleCode)
	}
	return data
}

func (h *ActHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

// ======================= JSON API =======================

// ListJSON возвращает список актов этапа + их приложения в JSON.
func (h *ActHandler) ListJSON(w http.ResponseWriter, r *http.Request) {
	stageGUID := chi.URLParam(r, "guid")
	contractID := chi.URLParam(r, "id")
	// Данные этапа — для шапки модалки.
	stage, _ := h.stageSvc.GetByID(r.Context(), stageGUID)

	acts, err := h.svc.ListByStage(r.Context(), stageGUID)
	if err != nil {
		log.Printf("act list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type attachJSON struct {
		GUID       string `json:"guid"`
		Kind       string `json:"kind"`
		KindName   string `json:"kind_name"`
		KindCustom string `json:"kind_custom"`
		Name       string `json:"name"`
		FilePath   string `json:"file_path"`
		Note       string `json:"note"`
	}

	type actJSON struct {
		GUID        string       `json:"guid"`
		ContractID  string       `json:"contract_id"`
		Kind        string       `json:"kind"`
		KindName    string       `json:"kind_name"`
		KindCustom  string       `json:"kind_custom"`
		Name        string       `json:"name"`
		Amount      float64      `json:"amount"`
		StageAmount float64      `json:"stage_amount"`
		Distributed float64      `json:"distributed"`
		SignDate    string       `json:"sign_date"`
		FilePath    string       `json:"file_path"`
		Note        string       `json:"note"`
		Attachments []attachJSON `json:"attachments"`
	}

	out := make([]actJSON, 0, len(acts))
	for _, a := range acts {
		signDate := ""
		if a.SignDate != nil {
			signDate = a.SignDate.Format("2006-01-02")
		}
		kindName := models.ActKindName(a.Kind)
		if a.Kind == models.ActKindCustom && a.KindCustom != "" {
			kindName = a.KindCustom
		}

		// Сколько из этого акта идёт на текущий этап.
		stageAmount, _ := h.svc.AmountByActAndStage(r.Context(), a.GUID, stageGUID)
		// Сколько всего из акта распределено по всем этапам.
		distributedAmount, _ := h.svc.SumByAct(r.Context(), a.GUID)

		// Подгружаем приложения.
		atts, _ := h.svc.ListAttachmentsByAct(r.Context(), a.GUID)
		attOut := make([]attachJSON, 0, len(atts))
		for _, at := range atts {
			atKindName := models.ActKindName(at.Kind)
			if at.Kind == models.ActKindCustom && at.KindCustom != "" {
				atKindName = at.KindCustom
			}
			attOut = append(attOut, attachJSON{
				GUID:       at.GUID,
				Kind:       at.Kind,
				KindName:   atKindName,
				KindCustom: at.KindCustom,
				Name:       at.Name,
				FilePath:   at.FilePath,
				Note:       at.Note,
			})
		}

		out = append(out, actJSON{
			GUID:        a.GUID,
			ContractID:  contractID,
			Kind:        a.Kind,
			KindName:    kindName,
			KindCustom:  a.KindCustom,
			Name:        a.Name,
			Amount:      a.Amount,
			StageAmount: stageAmount,
			Distributed: distributedAmount,
			SignDate:    signDate,
			FilePath:    a.FilePath,
			Note:        a.Note,
			Attachments: attOut,
		})
	}

	// Сумма актов этапа
	var total float64
	for _, a := range acts {
		stageAmount, _ := h.svc.AmountByActAndStage(r.Context(), a.GUID, stageGUID)
		total += stageAmount
	}

	// Шапка модалки
	type headerJSON struct {
		StageGUID          string  `json:"stage_guid"`
		ClosedByActs       float64 `json:"closed_by_acts"`
		ClosedByActsManual bool    `json:"closed_by_acts_manual"`
		Total              float64 `json:"total"`
		VATPercent         float64 `json:"vat_percent"`
	}

	var header headerJSON
	if stage != nil {
		header = headerJSON{
			StageGUID:          stage.GUID,
			ClosedByActs:       stage.ClosedByActs,
			ClosedByActsManual: stage.ClosedByActsManual,
			Total:              total,
			VATPercent:         stage.VATPercent,
		}
	}

	resp := map[string]any{
		"header": header,
		"acts":   out,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SumJSON — сумма актов по этапу.
func (h *ActHandler) SumJSON(w http.ResponseWriter, r *http.Request) {
	stageGUID := chi.URLParam(r, "guid")
	sum, err := h.svc.SumByStage(r.Context(), stageGUID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]float64{"amount": sum})
}

// ======================= Формы акта =======================

func (h *ActHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	stageGUID := chi.URLParam(r, "guid")

	stage, err := h.stageSvc.GetByID(r.Context(), stageGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	// Список этапов этого договора — для выбора, куда распределить сумму.
	stages, _ := h.stageSvc.List(r.Context(), contractID)

	data := h.commonData(r, "Новый акт")
	data["Contract"] = contract
	data["Stage"] = stage
	data["Stages"] = stages
	data["Act"] = &models.Act{}
	data["ActKinds"] = models.ActKindRefs()
	data["IsNew"] = true
	data["Error"] = ""
	h.render(w, "acts_form.html", data)
}

func (h *ActHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	actGUID := chi.URLParam(r, "actGuid")

	act, err := h.svc.GetByID(r.Context(), actGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	stages, _ := h.stageSvc.List(r.Context(), contractID)
	links, _ := h.svc.StagesLinkedToAct(r.Context(), actGUID)

	data := h.commonData(r, "Акт")
	data["Contract"] = contract
	data["Act"] = act
	data["Stages"] = stages
	data["ActKinds"] = models.ActKindRefs()
	data["Links"] = links
	data["IsNew"] = false
	data["Error"] = ""
	h.render(w, "acts_form.html", data)
}

// Create создаёт акт.
func (h *ActHandler) Create(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	stageGUID := chi.URLParam(r, "guid")

	act := parseActForm(r)

	// Если распределение свёрнуто (distribution_expanded пусто) —
	// привязываем всю сумму акта к текущему этапу.
	var links []models.ActStageLink
	if act.DistributionExpanded {
		links = parseStageLinks(r)
	}
	if len(links) == 0 {
		links = []models.ActStageLink{
			{StageGUID: stageGUID, Amount: act.Amount},
		}
	}

	if err := h.svc.Create(r.Context(), act, links); err != nil {
		stage, _ := h.stageSvc.GetByID(r.Context(), stageGUID)
		contract, _ := h.contractSvc.GetByID(r.Context(), contractID)
		stages, _ := h.stageSvc.List(r.Context(), contractID)
		data := h.commonData(r, "Новый акт")
		data["Contract"] = contract
		data["Stage"] = stage
		data["Stages"] = stages
		data["Act"] = act
		data["ActKinds"] = models.ActKindRefs()
		data["IsNew"] = true
		data["Error"] = humanizeActError(err)
		data["Links"] = parseStageLinks(r)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "acts_form.html", data)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/stages", http.StatusSeeOther)
}

// Update обновляет акт.
func (h *ActHandler) Update(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	actGUID := chi.URLParam(r, "actGuid")

	act := parseActForm(r)
	act.GUID = actGUID

	// Логика распределения:
	//   - Развёрнуто → берём links из формы.
	//   - Свёрнуто   → вся сумма акта идёт на текущий этап.
	//     Определяем «текущий» этап: если акт был привязан к этапу —
	//     оставляем его, иначе берём этап, с которого открыли форму (первый link).
	var links []models.ActStageLink
	if act.DistributionExpanded {
		links = parseStageLinks(r)
	}
	if len(links) == 0 {
		// Определяем текущий этап: первый, к которому был привязан акт.
		stageGUID, _ := h.svc.StageGUIDByAct(r.Context(), actGUID)
		if stageGUID != "" {
			links = []models.ActStageLink{
				{StageGUID: stageGUID, Amount: act.Amount},
			}
		}
	}

	if err := h.svc.Update(r.Context(), act, links); err != nil {
		contract, _ := h.contractSvc.GetByID(r.Context(), contractID)
		stages, _ := h.stageSvc.List(r.Context(), contractID)
		data := h.commonData(r, "Акт")
		data["Contract"] = contract
		data["Act"] = act
		data["Stages"] = stages
		data["ActKinds"] = models.ActKindRefs()
		data["IsNew"] = false
		data["Error"] = humanizeActError(err)
		data["Links"] = parseStageLinks(r)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "acts_form.html", data)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/stages", http.StatusSeeOther)
}

// parseStageLinks читает распределение по этапам из формы.
// Поля: stage_guid[] и link_amount[] — массивы одинаковой длины.
func parseStageLinks(r *http.Request) []models.ActStageLink {
	_ = r.ParseForm()

	stages := r.Form["stage_guid"]
	amounts := r.Form["link_amount"]

	n := len(stages)
	if len(amounts) < n {
		n = len(amounts)
	}

	var res []models.ActStageLink
	for i := 0; i < n; i++ {
		stageGUID := strings.TrimSpace(stages[i])
		if stageGUID == "" {
			continue
		}
		amt := parseMoneyFormValue(amounts[i])
		res = append(res, models.ActStageLink{
			StageGUID: stageGUID,
			Amount:    amt,
		})
	}
	return res
}

// Delete удаляет акт.
func (h *ActHandler) Delete(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	actGUID := chi.URLParam(r, "actGuid")

	if err := h.svc.Delete(r.Context(), actGUID); err != nil {
		log.Printf("act delete: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/stages", http.StatusSeeOther)
}

// OpenFile — открывает файл акта через pm://.
func (h *ActHandler) OpenFile(w http.ResponseWriter, r *http.Request) {
	actGUID := chi.URLParam(r, "actGuid")

	act, err := h.svc.GetByID(r.Context(), actGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if act.FilePath == "" {
		http.Error(w, "Файл акта не указан", http.StatusNotFound)
		return
	}
	h.renderPmOpen(w, act.FilePath)
}

// ======================= Формы приложения =======================

func (h *ActHandler) NewAttachmentForm(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	actGUID := chi.URLParam(r, "actGuid")

	act, err := h.svc.GetByID(r.Context(), actGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	data := h.commonData(r, "Новое приложение")
	data["Contract"] = contract
	data["Act"] = act
	data["Attachment"] = &models.ActAttachment{}
	data["ActKinds"] = models.ActKindRefs()
	data["IsNew"] = true
	data["Error"] = ""
	h.render(w, "act_attachments_form.html", data)
}

func (h *ActHandler) CreateAttachment(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	actGUID := chi.URLParam(r, "actGuid")

	a := parseAttachmentForm(r)
	a.ActGUID = actGUID

	if err := h.svc.CreateAttachment(r.Context(), a); err != nil {
		act, _ := h.svc.GetByID(r.Context(), actGUID)
		contract, _ := h.contractSvc.GetByID(r.Context(), contractID)
		data := h.commonData(r, "Новое приложение")
		data["Contract"] = contract
		data["Act"] = act
		data["Attachment"] = a
		data["ActKinds"] = models.ActKindRefs()
		data["IsNew"] = true
		data["Error"] = humanizeActError(err)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "act_attachments_form.html", data)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/stages", http.StatusSeeOther)
}

func (h *ActHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	attGUID := chi.URLParam(r, "attGuid")

	if err := h.svc.DeleteAttachment(r.Context(), attGUID); err != nil {
		log.Printf("attachment delete: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/stages", http.StatusSeeOther)
}

// OpenAttachment — открывает файл приложения через pm://.
func (h *ActHandler) OpenAttachment(w http.ResponseWriter, r *http.Request) {
	attGUID := chi.URLParam(r, "attGuid")

	att, err := h.svc.GetAttachmentByID(r.Context(), attGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if att.FilePath == "" {
		http.Error(w, "Файл не указан", http.StatusNotFound)
		return
	}
	h.renderPmOpen(w, att.FilePath)
}

// ======================= Helpers =======================

// renderPmOpen отдаёт HTML со скриптом, открывающим папку или файл через pm://.
func (h *ActHandler) renderPmOpen(w http.ResponseWriter, path string) {
	target := "pm://open?path=" + path
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html><html><body>
<script>
    window.location.href = ` + strconv.Quote(target) + `;
    setTimeout(function(){ window.close(); }, 500);
</script>
</body></html>`))
}

func parseActForm(r *http.Request) *models.Act {
	_ = r.ParseForm()

	amount := parseMoneyFormValue(r.FormValue("amount"))

	return &models.Act{
		Kind:                 strings.TrimSpace(r.FormValue("kind")),
		KindCustom:           strings.TrimSpace(r.FormValue("kind_custom")),
		Name:                 strings.TrimSpace(r.FormValue("name")),
		Amount:               amount,
		SignDate:             services.ParseDateSafe(r.FormValue("sign_date")),
		FilePath:             strings.TrimSpace(r.FormValue("file_path")),
		Note:                 strings.TrimSpace(r.FormValue("note")),
		DistributionExpanded: r.FormValue("distribution_expanded") == "on",
	}
}

func parseAttachmentForm(r *http.Request) *models.ActAttachment {
	_ = r.ParseForm()
	return &models.ActAttachment{
		Kind:       strings.TrimSpace(r.FormValue("kind")),
		KindCustom: strings.TrimSpace(r.FormValue("kind_custom")),
		Name:       strings.TrimSpace(r.FormValue("name")),
		FilePath:   strings.TrimSpace(r.FormValue("file_path")),
		Note:       strings.TrimSpace(r.FormValue("note")),
	}
}

// parseMoneyFormValue парсит денежное поле (убирает пробелы, NBSP, запятую меняет на точку).
func parseMoneyFormValue(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.NewReplacer(" ", "", "\u00A0", "", "\t", "").Replace(s)
	s = strings.Replace(s, ",", ".", 1)

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func humanizeActError(err error) string {
	switch {
	case errors.Is(err, services.ErrActNameRequired):
		return "Наименование обязательно"
	case errors.Is(err, services.ErrActKindRequired):
		return "Выберите вид акта"
	case errors.Is(err, services.ErrActKindCustomName):
		return "Укажите название своего типа акта"
	case errors.Is(err, services.ErrActNegative):
		return "Суммы не могут быть отрицательными"
	case errors.Is(err, services.ErrActOverDistribute):
		return "Распределение по этапам превышает сумму акта"
	case errors.Is(err, services.ErrAttachmentRequired):
		return "Приложение должно быть привязано к акту"
	}
	return "Не удалось сохранить: " + err.Error()
}

// PickedPath возвращает путь, выбранный в трее через pm://pick-file.
func (h *ActHandler) PickedPath(w http.ResponseWriter, r *http.Request) {
	type result struct {
		Path  string `json:"path"`
		Mtime int64  `json:"mtime"`
	}

	appData := os.Getenv("APPDATA")
	if appData == "" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result{})
		return
	}

	path := filepath.Join(appData, "ProjectManager", "picked_path.txt")
	info, err := os.Stat(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result{})
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result{
		Path:  strings.TrimSpace(string(data)),
		Mtime: info.ModTime().UnixMilli(),
	})
}

// ToggleStageManual переключает режим «Закрыто по актам» у этапа.
func (h *ActHandler) ToggleStageManual(w http.ResponseWriter, r *http.Request) {
	stageGUID := chi.URLParam(r, "guid")

	stage, err := h.stageSvc.GetByID(r.Context(), stageGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Инвертируем режим
	stage.ClosedByActsManual = !stage.ClosedByActsManual

	// Если переключаем на «из актов» — сразу пересчитываем сумму.
	if !stage.ClosedByActsManual {
		sum, _ := h.svc.SumByStage(r.Context(), stageGUID)
		stage.ClosedByActs = sum
	}

	if err := h.stageSvc.Update(r.Context(), stage); err != nil {
		log.Printf("toggle stage manual: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"closed_by_acts":        stage.ClosedByActs,
		"closed_by_acts_manual": stage.ClosedByActsManual,
	})
}

// SetClosed сохраняет ручное значение «Закрыто по актам» для этапа.
// Используется из модалки актов.
func (h *ActHandler) SetClosed(w http.ResponseWriter, r *http.Request) {
	stageGUID := chi.URLParam(r, "guid")

	_ = r.ParseForm()
	raw := r.FormValue("closed_by_acts")

	value := parseMoneyFormValue(raw)

	stage, err := h.stageSvc.GetByID(r.Context(), stageGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	stage.ClosedByActs = value
	stage.ClosedByActsManual = true

	if err := h.stageSvc.Update(r.Context(), stage); err != nil {
		log.Printf("set closed by acts: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"closed_by_acts":        stage.ClosedByActs,
		"closed_by_acts_manual": stage.ClosedByActsManual,
	})
}

// EditAttachmentForm — форма редактирования приложения.
func (h *ActHandler) EditAttachmentForm(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	attGUID := chi.URLParam(r, "attGuid")

	att, err := h.svc.GetAttachmentByID(r.Context(), attGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)
	act, _ := h.svc.GetByID(r.Context(), att.ActGUID)

	data := h.commonData(r, "Приложение")
	data["Contract"] = contract
	data["Act"] = act
	data["Attachment"] = att
	data["ActKinds"] = models.ActKindRefs()
	data["IsNew"] = false
	data["Error"] = ""
	h.render(w, "act_attachments_form.html", data)
}

// UpdateAttachment — сохранение приложения.
func (h *ActHandler) UpdateAttachment(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	attGUID := chi.URLParam(r, "attGuid")

	a := parseAttachmentForm(r)
	a.GUID = attGUID

	// ActGUID оставляем тем же — приложение нельзя переносить между актами.
	existing, err := h.svc.GetAttachmentByID(r.Context(), attGUID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.ActGUID = existing.ActGUID

	if err := h.svc.UpdateAttachment(r.Context(), a); err != nil {
		act, _ := h.svc.GetByID(r.Context(), existing.ActGUID)
		contract, _ := h.contractSvc.GetByID(r.Context(), contractID)
		data := h.commonData(r, "Приложение")
		data["Contract"] = contract
		data["Act"] = act
		data["Attachment"] = a
		data["ActKinds"] = models.ActKindRefs()
		data["IsNew"] = false
		data["Error"] = humanizeActError(err)
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, "act_attachments_form.html", data)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/stages", http.StatusSeeOther)
}
