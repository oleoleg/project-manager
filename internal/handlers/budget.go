package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type BudgetHandler struct {
	svc         *services.BudgetService
	contractSvc *services.ContractService
	stageSvc    *services.StageService
	costSvc     *services.StageCostService
	cpSvc       *services.CounterpartyService
	tmpl        *template.Template
}

func NewBudgetHandler(svc *services.BudgetService, contractSvc *services.ContractService, stageSvc *services.StageService, costSvc *services.StageCostService, cpSvc *services.CounterpartyService, tmpl *template.Template) *BudgetHandler {
	return &BudgetHandler{svc: svc, contractSvc: contractSvc, stageSvc: stageSvc, costSvc: costSvc, cpSvc: cpSvc, tmpl: tmpl}
}

func (h *BudgetHandler) Index(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	editMode := r.URL.Query().Get("edit") == "1"

	settings, hasSettings, _ := h.svc.GetSettings(r.Context(), contractID)
	snapshots, _ := h.svc.GetAllSnapshots(r.Context(), contractID)

	selectedSnapshotID := r.URL.Query().Get("snapshot")
	var activeSnapshot *models.BudgetPlanSnapshot
	var lines []models.BudgetPlanLine

	if selectedSnapshotID != "" {
		activeSnapshot, lines, _ = h.svc.GetPlanByID(r.Context(), selectedSnapshotID)
	} else if len(snapshots) > 0 && !editMode {
		var targetID string
		for _, snap := range snapshots {
			if snap.IsActive {
				targetID = snap.ID
				break
			}
		}
		if targetID == "" {
			targetID = snapshots[0].ID
		}
		activeSnapshot, lines, _ = h.svc.GetPlanByID(r.Context(), targetID)
	}

	isReadOnly := false
	if activeSnapshot != nil && !editMode {
		isReadOnly = true
		if len(activeSnapshot.Settings) > 0 {
			var snapshotSettings models.BudgetSettings
			if err := json.Unmarshal(activeSnapshot.Settings, &snapshotSettings); err == nil {
				settings = snapshotSettings
			}
		}
	}

	// Получаем затраты для отображения в третьей вкладке
	var displayCosts []models.StageCost
	if activeSnapshot != nil && len(activeSnapshot.StageCostsSnapshot) > 0 {
		_ = json.Unmarshal(activeSnapshot.StageCostsSnapshot, &displayCosts)
	} else {
		displayCosts, _ = h.costSvc.GetAllCostsForContract(r.Context(), contractID)
	}

	var stages []models.Stage
	if activeSnapshot != nil && len(activeSnapshot.StageDataSnapshot) > 0 {
		_ = json.Unmarshal(activeSnapshot.StageDataSnapshot, &stages)
	} else {
		stages, _ = h.stageSvc.List(r.Context(), contractID)
	}
	stagesMap := make(map[string]models.Stage)
	for _, s := range stages {
		stagesMap[s.GUID] = s
	}
	workKinds, _ := h.stageSvc.WorkKinds(r.Context())
	// Собираем доступные виды работ для каждого этапа
	costs, _ := h.costSvc.GetAllCostsForContract(r.Context(), contractID)
	stageWorkKinds := make(map[string][]string)
	for _, c := range costs {
		found := false
		for _, wk := range stageWorkKinds[c.StageGUID] {
			if wk == c.WorkKindCode {
				found = true
				break
			}
		}
		if !found {
			stageWorkKinds[c.StageGUID] = append(stageWorkKinds[c.StageGUID], c.WorkKindCode)
		}
	}

	stageWorkKindsJSON, _ := json.Marshal(stageWorkKinds)
	workKindsJSON, _ := json.Marshal(workKinds)

	cp, _ := h.cpSvc.GetByID(r.Context(), contract.CounterpartyID)

	data := map[string]any{
		"Title":              "Бюджет договора",
		"User":               middleware.UserFromContext(r.Context()),
		"Contract":           contract,
		"Settings":           settings,
		"HasSettings":        hasSettings,
		"Snapshots":          snapshots,
		"Snapshot":           activeSnapshot,
		"PlanLines":          lines,
		"Flash":              r.URL.Query().Get("flash"),
		"WorkKinds":          workKinds,
		"IsReadOnly":         isReadOnly,
		"EditMode":           editMode,
		"Counterparty":       cp,
		"Costs":              displayCosts,
		"StagesMap":          stagesMap,
		"StageWorkKindsJSON": string(stageWorkKindsJSON),
		"WorkKindsJSON":      string(workKindsJSON),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_plan.html", data)
}

func (h *BudgetHandler) Generate(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	u := middleware.UserFromContext(r.Context())

	_ = r.ParseForm()
	note := r.FormValue("note")

	// 1. СОХРАНЯЕМ НАСТРОЙКИ
	startDate, _ := time.Parse("2006-01-02", r.FormValue("project_start_date"))
	endDateStr := r.FormValue("project_end_date")
	var endDate *time.Time
	if endDateStr != "" {
		if d, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &d
		}
	}
	delayDays, _ := strconv.Atoi(r.FormValue("payment_delay_days"))
	equipPct, _ := strconv.ParseFloat(r.FormValue("equip_payment_percent"), 64)
	equipDelay, _ := strconv.Atoi(r.FormValue("equip_payment_delay"))
	overhead, _ := strconv.ParseFloat(r.FormValue("overhead_rate"), 64)

	settings := models.BudgetSettings{
		ContractID:          contractID,
		ProjectStartDate:    startDate,
		ProjectEndDate:      endDate,
		PaymentDelayDays:    delayDays,
		EquipPaymentPercent: equipPct,
		EquipPaymentDelay:   equipDelay,
		OverheadRate:        overhead,
	}

	if err := h.svc.SaveSettings(r.Context(), settings); err != nil {
		log.Printf("save settings before generate: %v", err)
		http.Error(w, "Ошибка сохранения настроек: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. ПАРСИМ И СОХРАНЯЕМ ТОЛЬКО ЗАТРАТЫ
	formCosts := parseCostsFromBudgetForm(r)

	costsByStage := make(map[string][]models.StageCost)
	for _, c := range formCosts {
		costsByStage[c.StageGUID] = append(costsByStage[c.StageGUID], c)
	}

	for stageGUID, stageCosts := range costsByStage {
		if err := h.costSvc.ReplaceForStage(r.Context(), stageGUID, stageCosts); err != nil {
			log.Printf("save stage costs before generate: %v", err)
			http.Error(w, "Ошибка сохранения затрат: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 3. ГЕНЕРИРУЕМ ПЛАН
	_, err := h.svc.GeneratePlan(r.Context(), contractID, u.ID, note)
	if err != nil {
		log.Printf("generate plan: %v", err)
		http.Error(w, "Ошибка: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget", http.StatusSeeOther)
}

// NewVersion отображает страницу создания новой версии плана
func (h *BudgetHandler) NewVersion(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	settings, _, _ := h.svc.GetSettings(r.Context(), contractID)
	costs, _ := h.costSvc.GetAllCostsForContract(r.Context(), contractID)
	stages, _ := h.stageSvc.List(r.Context(), contractID)
	workKinds, _ := h.stageSvc.WorkKinds(r.Context())
	workKindsJSON, _ := json.Marshal(workKinds)

	type CostRow struct {
		IsHeader bool
		Stage    models.Stage
		Cost     models.StageCost
	}
	var displayRows []CostRow

	for _, s := range stages {
		displayRows = append(displayRows, CostRow{
			IsHeader: true,
			Stage:    s,
		})

		hasCosts := false
		for _, c := range costs {
			if c.StageGUID == s.GUID {
				displayRows = append(displayRows, CostRow{
					IsHeader: false,
					Stage:    s,
					Cost:     c,
				})
				hasCosts = true
			}
		}

		if !hasCosts {
			displayRows = append(displayRows, CostRow{
				IsHeader: false,
				Stage:    s,
				Cost: models.StageCost{
					StageGUID:    s.GUID,
					WorkKindCode: s.WorkKindCode,
				},
			})
		}
	}

	cp, _ := h.cpSvc.GetByID(r.Context(), contract.CounterpartyID)

	data := map[string]any{
		"Title":         "Новая версия бюджета",
		"User":          middleware.UserFromContext(r.Context()),
		"Contract":      contract,
		"Counterparty":  cp,
		"Settings":      settings,
		"DisplayRows":   displayRows,
		"WorkKinds":     workKinds,
		"Stages":        stages,
		"WorkKindsJSON": string(workKindsJSON),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_new.html", data)
}

func (h *BudgetHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	_ = r.ParseForm()

	startDate, _ := time.Parse("2006-01-02", r.FormValue("project_start_date"))
	endDateStr := r.FormValue("project_end_date")

	var endDate *time.Time
	if endDateStr != "" {
		if d, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &d
		}
	}

	delayDays, _ := strconv.Atoi(r.FormValue("payment_delay_days"))
	equipPct, _ := strconv.ParseFloat(r.FormValue("equip_payment_percent"), 64)
	equipDelay, _ := strconv.Atoi(r.FormValue("equip_payment_delay"))
	overhead, _ := strconv.ParseFloat(r.FormValue("overhead_rate"), 64)

	s := models.BudgetSettings{
		ContractID:          contractID,
		ProjectStartDate:    startDate,
		ProjectEndDate:      endDate,
		PaymentDelayDays:    delayDays,
		EquipPaymentPercent: equipPct,
		EquipPaymentDelay:   equipDelay,
		OverheadRate:        overhead,
	}

	if err := h.svc.SaveSettings(r.Context(), s); err != nil {
		log.Printf("save settings: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget?flash=Настройки+сохранены", http.StatusSeeOther)
}

func (h *BudgetHandler) DeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	snapshotID := chi.URLParam(r, "snapshotID")

	if err := h.svc.DeleteSnapshot(r.Context(), snapshotID); err != nil {
		log.Printf("delete snapshot: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/budget?flash=Снимок+удален", http.StatusSeeOther)
}

func (h *BudgetHandler) ActivateSnapshot(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	snapshotID := chi.URLParam(r, "snapshotID")

	// Восстанавливаем данные из снимка
	if err := h.svc.RestoreFromSnapshot(r.Context(), contractID, snapshotID); err != nil {
		log.Printf("restore from snapshot: %v", err)
		http.Error(w, "Ошибка восстановления данных из снимка", http.StatusInternalServerError)
		return
	}

	// Делаем его активным
	if err := h.svc.SetActiveSnapshot(r.Context(), contractID, snapshotID); err != nil {
		log.Printf("activate snapshot: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/budget?flash=Снимок+активирован+и+данные+восстановлены", http.StatusSeeOther)
}

func (h *BudgetHandler) ApproveSnapshot(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	snapshotID := chi.URLParam(r, "snapshotID")
	u := middleware.UserFromContext(r.Context())

	if err := h.svc.ApproveSnapshot(r.Context(), contractID, snapshotID, u.ID); err != nil {
		log.Printf("approve snapshot: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/budget?snapshot="+snapshotID, http.StatusSeeOther)
}

func (h *BudgetHandler) UnapproveSnapshot(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	snapshotID := chi.URLParam(r, "snapshotID")

	if err := h.svc.UnapproveSnapshot(r.Context(), snapshotID); err != nil {
		log.Printf("unapprove snapshot: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/budget?snapshot="+snapshotID, http.StatusSeeOther)
}

// FactIndex отображает страницу ФинФакта.
func (h *BudgetHandler) FactIndex(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)
	stages, _ := h.stageSvc.List(r.Context(), contractID)
	stagesMap := make(map[string]models.Stage)
	for _, s := range stages {
		stagesMap[s.GUID] = s
	}

	lines, err := h.svc.GetFactLines(r.Context(), contractID)
	if err != nil {
		log.Printf("get fact lines: %v", err)
		lines = nil
	}

	groupedLines := make(map[string][]models.BudgetFactLine)
	var groupOrder []string
	orderMap := map[string]int{
		"Платежи по договору": 1, "Закупка по договору": 2, "Зарплата сотрудникам": 3,
		"Накладные": 4, "Командировки": 5, "Субподрядчики": 6, "Комиссия": 7, "Другое (расходы)": 8,
	}

	for _, l := range lines {
		if _, exists := groupedLines[l.OperationType]; !exists {
			groupOrder = append(groupOrder, l.OperationType)
		}
		groupedLines[l.OperationType] = append(groupedLines[l.OperationType], l)
	}

	sort.Slice(groupOrder, func(i, j int) bool {
		return orderMap[groupOrder[i]] < orderMap[groupOrder[j]]
	})

	checkData, _ := h.svc.GetFactCheckData(r.Context(), contractID)
	var factPayments float64
	for _, l := range lines {
		if l.OperationType == "Платежи по договору" && l.IsConfirmed {
			factPayments += l.FactIncome
		}
	}

	warnings := []string{}
	if math.Abs(factPayments-checkData["total_acts"]) > 1.0 && factPayments > 0 {
		warnings = append(warnings, fmt.Sprintf("Внимание: Сумма платежей по факту (%.2f ₽) не сходится с суммой закрытых актов в договоре (%.2f ₽).", factPayments, checkData["total_acts"]))
	}

	snapshots, _ := h.svc.GetAllSnapshots(r.Context(), contractID)
	hasActivePlan := false
	for _, snap := range snapshots {
		if snap.IsActive {
			hasActivePlan = true
			break
		}
	}
	workKinds, _ := h.stageSvc.WorkKinds(r.Context())
	// Собираем доступные виды работ для каждого этапа
	costs, _ := h.costSvc.GetAllCostsForContract(r.Context(), contractID)
	stageWorkKinds := make(map[string][]string)
	for _, c := range costs {
		found := false
		for _, wk := range stageWorkKinds[c.StageGUID] {
			if wk == c.WorkKindCode {
				found = true
				break
			}
		}
		if !found {
			stageWorkKinds[c.StageGUID] = append(stageWorkKinds[c.StageGUID], c.WorkKindCode)
		}
	}

	stageWorkKindsJSON, _ := json.Marshal(stageWorkKinds)
	workKindsJSON, _ := json.Marshal(workKinds)

	cp, _ := h.cpSvc.GetByID(r.Context(), contract.CounterpartyID)

	data := map[string]any{
		"Title":              "ФинФакт договора",
		"User":               middleware.UserFromContext(r.Context()),
		"Contract":           contract,
		"Counterparty":       cp,
		"FactLines":          lines,
		"GroupedLines":       groupedLines,
		"GroupOrder":         groupOrder,
		"HasActivePlan":      hasActivePlan,
		"WorkKinds":          workKinds,
		"Flash":              r.URL.Query().Get("flash"),
		"Warnings":           warnings,
		"Stages":             stages,
		"StageWorkKindsJSON": string(stageWorkKindsJSON),
		"WorkKindsJSON":      string(workKindsJSON),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_fact.html", data)
}

func (h *BudgetHandler) FactGenerate(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")

	if err := h.svc.GenerateFactFromPlan(r.Context(), contractID); err != nil {
		log.Printf("generate fact: %v", err)
		http.Error(w, "Ошибка генерации факта: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact?flash=Факт+успешно+сформирован", http.StatusSeeOther)
}

func (h *BudgetHandler) FactDelete(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")

	if err := h.svc.DeleteFact(r.Context(), contractID); err != nil {
		log.Printf("delete fact: %v", err)
		http.Error(w, "Ошибка удаления факта", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact?flash=Факт+удален", http.StatusSeeOther)
}

func (h *BudgetHandler) FactUpdate(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	factID := chi.URLParam(r, "factID")
	_ = r.ParseForm()

	factIncome, _ := strconv.ParseFloat(r.FormValue("fact_income"), 64)
	factExpense, _ := strconv.ParseFloat(r.FormValue("fact_expense"), 64)
	isConfirmed := r.FormValue("is_confirmed") == "on"

	if err := h.svc.UpdateFactLine(r.Context(), factID, factIncome, factExpense, isConfirmed); err != nil {
		log.Printf("update fact line: %v", err)
		http.Error(w, "Ошибка сохранения", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact", http.StatusSeeOther)
}

func (h *BudgetHandler) FactSaveBatch(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	log.Printf("ПОЛУЧЕН ЗАПРОС НА СОХРАНЕНИЕ ФАКТА ДЛЯ %s", contractID)

	var payload struct {
		Lines []struct {
			ID           string  `json:"id"`
			FactIncome   float64 `json:"fact_income"`
			FactExpense  float64 `json:"fact_expense"`
			IsConfirmed  bool    `json:"is_confirmed"`
			IsManualEdit bool    `json:"is_manual_edit"`
		} `json:"lines"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("ОШИБКА ДЕКОДИРОВАНИЯ JSON: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var updates []db.FactUpdateData
	for _, l := range payload.Lines {
		updates = append(updates, db.FactUpdateData{
			ID:           l.ID,
			FactIncome:   l.FactIncome,
			FactExpense:  l.FactExpense,
			IsConfirmed:  l.IsConfirmed,
			IsManualEdit: l.IsManualEdit,
		})
	}

	if err := h.svc.UpdateFactLinesBatch(r.Context(), updates); err != nil {
		log.Printf("ОШИБКА БД ПРИ СОХРАНЕНИИ: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *BudgetHandler) FactSaveForm(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	_ = r.ParseForm()
	log.Println("=== FactSaveForm CALLED ===")
	log.Printf("DEBUG: ids=%d, stageGUIDs=%d", len(r.Form["fact_id[]"]), len(r.Form["stage_guid[]"]))
	for i := 0; i < len(r.Form["fact_id[]"]); i++ {
		log.Printf("DEBUG[%d]: id='%s', stage_guid='%s'", i, r.Form["fact_id[]"][i], r.Form["stage_guid[]"][i])
	}

	ids := r.Form["fact_id[]"]
	names := r.Form["name[]"]
	opDates := r.Form["operation_date[]"]
	delayDays := r.Form["delay_days[]"]
	payDates := r.Form["payment_date[]"]
	incIncomes := r.Form["fact_income[]"]
	incExpenses := r.Form["fact_expense[]"]
	workKindCodes := r.Form["work_kind_code[]"]
	opTypes := r.Form["operation_type[]"]
	stageGUIDs := r.Form["stage_guid[]"]
	log.Printf("DEBUG: Received %d fact_ids, %d stage_guids", len(ids), len(stageGUIDs))
	for i := 0; i < len(ids); i++ {
		id := ids[i]
		sg := ""
		if i < len(stageGUIDs) {
			sg = stageGUIDs[i]
		}
		log.Printf("DEBUG[%d]: fact_id='%s', stage_guid='%s'", i, id, sg)
	}
	n := len(ids)

	pad := func(arr []string) []string {
		if len(arr) >= n {
			return arr[:n]
		}
		out := make([]string, n)
		copy(out, arr)
		return out
	}

	confirmedSet := make(map[string]bool)
	for _, id := range r.Form["is_confirmed[]"] {
		confirmedSet[id] = true
	}
	confirms := make([]string, n)
	for i, id := range ids {
		if confirmedSet[id] {
			confirms[i] = id
		}
	}

	for i := 0; i < len(ids); i++ {
		log.Printf("DEBUG[%d]: work_kind_code='%s'", i, workKindCodes[i])
	}

	err := h.svc.BatchUpdateFactLines(
		r.Context(), contractID,
		ids, pad(names), pad(workKindCodes), pad(opTypes), pad(stageGUIDs),
		pad(opDates), pad(delayDays), pad(payDates),
		pad(incIncomes), pad(incExpenses), confirms,
	)
	if err != nil {
		log.Printf("ОШИБКА МАССОВОГО ОБНОВЛЕНИЯ: %v", err)
		http.Error(w, "Ошибка сохранения: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact?flash=Изменения+сохранены", http.StatusSeeOther)
}

// FactAutoConfirmAutoRows квитирует ЗП и Накладные, если месяц наступил
func (h *BudgetHandler) FactAutoConfirmAutoRows(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")

	// Просто вызываем метод репозитория (создадим его на следующем шаге)
	if err := h.svc.AutoConfirmSalaryAndOverhead(r.Context(), contractID); err != nil {
		log.Printf("auto confirm: %v", err)
		http.Error(w, "Ошибка автоквитирования: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact?flash=ЗП+и+Накладные+успешно+квитированы", http.StatusSeeOther)
}

// parseCostsFromBudgetForm собирает затраты из формы создания новой версии
func parseCostsFromBudgetForm(r *http.Request) []models.StageCost {
	_ = r.ParseForm()

	stageGUIDs := r.Form["cost_stage_guid[]"]
	workKinds := r.Form["cost_work_kind[]"]
	amounts := r.Form["cost_amount[]"]
	fotPlans := r.Form["cost_fot_plan[]"]
	days := r.Form["cost_labor_days[]"]
	rates := r.Form["cost_rate_per_day[]"]
	travels := r.Form["cost_travel[]"]
	eqSales := r.Form["cost_equipment_sale[]"]
	eqCosts := r.Form["cost_equipment_cost[]"]
	subs := r.Form["cost_subcontract[]"]
	comms := r.Form["cost_commission[]"]
	others := r.Form["cost_other[]"]

	n := len(stageGUIDs)
	res := make([]models.StageCost, 0, n)

	getStr := func(arr []string, i int) string {
		if i < len(arr) {
			return arr[i]
		}
		return ""
	}

	parseMoney := func(s string) float64 {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, ",", ".")
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	for i := 0; i < n; i++ {
		wc := strings.TrimSpace(getStr(workKinds, i))
		if wc == "" {
			continue
		}
		res = append(res, models.StageCost{
			StageGUID:      getStr(stageGUIDs, i),
			WorkKindCode:   wc,
			Cost:           parseMoney(getStr(amounts, i)),
			FOTPlan:        parseMoney(getStr(fotPlans, i)),
			LaborDays:      parseMoney(getStr(days, i)),
			RatePerDay:     parseMoney(getStr(rates, i)),
			TravelExpenses: parseMoney(getStr(travels, i)),
			EquipmentSale:  parseMoney(getStr(eqSales, i)),
			EquipmentCost:  parseMoney(getStr(eqCosts, i)),
			Subcontract:    parseMoney(getStr(subs, i)),
			Commission:     parseMoney(getStr(comms, i)),
			OtherExpenses:  parseMoney(getStr(others, i)),
		})
	}
	return res
}

func (h *BudgetHandler) GetSummaryChart(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	selectedSnapshotID := r.URL.Query().Get("snapshot")

	var lines []models.BudgetPlanLine

	if selectedSnapshotID != "" {
		_, lines, _ = h.svc.GetPlanByID(r.Context(), selectedSnapshotID)
	} else {
		_, lines, _ = h.svc.GetLatestPlan(r.Context(), contractID)
	}

	expenseSums := make(map[string]float64)
	deptSums := make(map[string]float64)

	totalIncome := 0.0
	totalExpense := 0.0

	for _, l := range lines {
		totalIncome += l.Income
		totalExpense += l.Expense

		if l.OperationType != "Платежи по договору" && l.Expense > 0 {
			expenseSums[l.OperationType] += l.Expense
			deptSums[l.WorkKindCode] += l.Expense
		}
	}

	profit := totalIncome - totalExpense
	if profit != 0 {
		expenseSums["Прибыль"] = profit
	}

	resp := struct {
		ExpensesChart    ChartData `json:"expenses_chart"`
		DepartmentsChart ChartData `json:"departments_chart"`
	}{
		ExpensesChart:    ChartData{Labels: []string{}, Data: []float64{}},
		DepartmentsChart: ChartData{Labels: []string{}, Data: []float64{}},
	}

	for k, v := range expenseSums {
		resp.ExpensesChart.Labels = append(resp.ExpensesChart.Labels, k)
		resp.ExpensesChart.Data = append(resp.ExpensesChart.Data, v)
	}

	for k, v := range deptSums {
		resp.DepartmentsChart.Labels = append(resp.DepartmentsChart.Labels, k)
		resp.DepartmentsChart.Data = append(resp.DepartmentsChart.Data, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type ChartData struct {
	Labels []string  `json:"labels"`
	Data   []float64 `json:"data"`
}

func (h *BudgetHandler) PlanFactIndex(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	// Определяем режим (по умолчанию - только подтвержденные)
	factMode := r.URL.Query().Get("fact")
	if factMode == "" {
		factMode = "confirmed"
	}
	onlyConfirmed := (factMode == "confirmed")

	summary, err := h.svc.GetPlanFactSummary(r.Context(), contractID, onlyConfirmed)
	if err != nil {
		log.Printf("get plan fact summary: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	workKinds, _ := h.stageSvc.WorkKinds(r.Context())

	// Группируем данные по видам работ для графика и итогов
	type WorkKindSummary struct {
		Code       string
		Name       string
		PlanProfit float64
		FactProfit float64
		Deviation  float64
	}

	// Считаем прибыль по видам работ (Сумма всех Балансов по этому виду работ)
	wkMap := make(map[string]*WorkKindSummary)
	for _, row := range summary {
		if _, ok := wkMap[row.WorkKindCode]; !ok {
			name := row.WorkKindCode
			for _, wk := range workKinds {
				if wk.Code == row.WorkKindCode {
					name = wk.Name
					break
				}
			}
			wkMap[row.WorkKindCode] = &WorkKindSummary{Code: row.WorkKindCode, Name: name}
		}
		wkMap[row.WorkKindCode].PlanProfit += row.PlanBalance
		wkMap[row.WorkKindCode].FactProfit += row.FactBalance
		wkMap[row.WorkKindCode].Deviation += row.Deviation
	}

	// Сортируем виды работ (по порядку как в Excel: ПИР, ППО, СМР, ПНР, Сборка, Продажа)
	order := map[string]int{"pir": 1, "ppo": 2, "smr": 3, "pnr": 4, "sbor": 5, "sale": 6}

	var wkList []*WorkKindSummary
	for _, v := range wkMap {
		wkList = append(wkList, v)
	}

	sort.Slice(wkList, func(i, j int) bool {
		// Получаем вес (если кода нет в списке, ставим 99)
		w1, ok1 := order[wkList[i].Code]
		if !ok1 {
			w1 = 99
		}

		w2, ok2 := order[wkList[j].Code]
		if !ok2 {
			w2 = 99
		}

		// Если веса разные — сортируем по весу
		if w1 != w2 {
			return w1 < w2
		}
		// Если веса одинаковые (например, оба 99) — сортируем по алфавиту (имени)
		return wkList[i].Name < wkList[j].Name
	})

	data := map[string]any{
		"Title":       "Сравнение План vs Факт",
		"User":        middleware.UserFromContext(r.Context()),
		"Contract":    contract,
		"Summary":     summary,
		"WorkKinds":   workKinds,
		"GroupedByWK": wkList,
		"FactMode":    factMode,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_compare.html", data)
}
