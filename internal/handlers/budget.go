package handlers

import (
	"context"
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
	contract, err := h.contractSvc.GetByID(r.Context(), contractID)
	if err != nil || contract == nil {
		log.Printf("budget index get contract %s: %v", contractID, err)
		http.Error(w, "Договор не найден", http.StatusNotFound)
		return
	}

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

	// Собираем доступные виды работ для каждого этапа через хелпер
	stageWorkKinds := h.buildStageWorkKinds(r.Context(), contractID, stages)

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
	settings := parseSettingsFromForm(r, contractID)

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
	contract, err := h.contractSvc.GetByID(r.Context(), contractID)
	if err != nil || contract == nil {
		log.Printf("budget new version get contract %s: %v", contractID, err)
		http.Error(w, "Договор не найден", http.StatusNotFound)
		return
	}

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
					StageGUID: s.GUID,
					WorkKind:  s.WorkKind,
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

	s := parseSettingsFromForm(r, contractID)

	if err := h.svc.SaveSettings(r.Context(), s); err != nil {
		log.Printf("save settings: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget?flash=Настройки+сохранены", http.StatusSeeOther)
}

// DeleteSnapshot удаляет снимок (факт при этом не трогаем, так как строки плана не удаляются)
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

// ActivateSnapshot делает снимок активным и восстанавливает данные
func (h *BudgetHandler) ActivateSnapshot(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	snapshotID := chi.URLParam(r, "snapshotID")

	// Восстанавливаем данные из снимка (этапы, затраты, строки плана)
	if err := h.svc.RestoreFromSnapshot(r.Context(), contractID, snapshotID); err != nil {
		log.Printf("restore from snapshot: %v", err)
		http.Error(w, "Ошибка восстановления данных из снимка: "+err.Error(), http.StatusInternalServerError)
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
	contract, err := h.contractSvc.GetByID(r.Context(), contractID)
	if err != nil || contract == nil {
		log.Printf("budget fact get contract %s: %v", contractID, err)
		http.Error(w, "Договор не найден", http.StatusNotFound)
		return
	}

	stages, err := h.stageSvc.List(r.Context(), contractID)
	if err != nil {
		log.Printf("budget fact get stages %s: %v", contractID, err)
		http.Error(w, "Ошибка загрузки этапов", http.StatusInternalServerError)
		return
	}

	stagesMap := make(map[string]models.Stage)
	for _, s := range stages {
		stagesMap[s.GUID] = s
	}

	lines, err := h.svc.GetFactLines(r.Context(), contractID)
	if err != nil {
		log.Printf("get fact lines: %v", err)
		lines = nil
	}

	// Справочник порядка групп (чтобы "Платежи" были выше "Зарплаты")
	orderMap := map[string]int{
		"Платежи по договору": 1, "Закупка по договору": 2, "Зарплата сотрудникам": 3,
		"Накладные": 4, "Командировки": 5, "Субподрядчики": 6, "Комиссия": 7, "Другое (расходы)": 8,
	}

	groupedLines := make(map[string][]models.BudgetFactLine)
	seenTypes := make(map[string]bool)
	var groupOrder []string

	for _, l := range lines {
		opType := l.OperationType
		if !seenTypes[opType] {
			seenTypes[opType] = true
			groupOrder = append(groupOrder, opType)
		}
		groupedLines[opType] = append(groupedLines[opType], l)
	}

	// Безопасная сортировка: если тип операции неизвестен, ставим его в конец (99)
	sort.Slice(groupOrder, func(i, j int) bool {
		w1, ok1 := orderMap[groupOrder[i]]
		if !ok1 {
			w1 = 99
		}
		w2, ok2 := orderMap[groupOrder[j]]
		if !ok2 {
			w2 = 99
		}
		return w1 < w2
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

	// Собираем доступные виды работ для каждого этапа через хелпер
	stageWorkKinds := h.buildStageWorkKinds(r.Context(), contractID, stages)

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

	ids := r.Form["fact_id[]"]
	names := r.Form["name[]"]
	opDates := r.Form["operation_date[]"]
	delayDays := r.Form["delay_days[]"]
	payDates := r.Form["payment_date[]"]
	incIncomes := r.Form["fact_income[]"]
	incExpenses := r.Form["fact_expense[]"]
	workKinds := r.Form["work_kind[]"]
	opTypes := r.Form["operation_type[]"]
	stageNumbers := r.Form["stage_number[]"]
	stageNames := r.Form["stage_name[]"]

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

	err := h.svc.BatchUpdateFactLines(
		r.Context(), contractID,
		ids, pad(names), pad(workKinds), pad(opTypes), pad(stageNumbers), pad(stageNames),
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

	// Локальная обертка, чтобы не обрабатывать ошибку каждый раз (при ошибке вернет 0)
	parseMoney := func(s string) float64 {
		v, _ := services.ParseMoney(s)
		return v
	}

	for i := 0; i < n; i++ {
		wc := strings.TrimSpace(getStr(workKinds, i))
		if wc == "" {
			continue
		}
		res = append(res, models.StageCost{
			StageGUID:      getStr(stageGUIDs, i),
			WorkKind:       wc,
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

// parseSettingsFromForm собирает настройки бюджета из формы
func parseSettingsFromForm(r *http.Request, contractID string) models.BudgetSettings {
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

	return models.BudgetSettings{
		ContractID:          contractID,
		ProjectStartDate:    startDate,
		ProjectEndDate:      endDate,
		PaymentDelayDays:    delayDays,
		EquipPaymentPercent: equipPct,
		EquipPaymentDelay:   equipDelay,
		OverheadRate:        overhead,
	}
}

// buildStageWorkKinds собирает список доступных видов работ для каждого этапа
// по его номеру (StageNumber). Возвращает map[НомерЭтапа][]КодВидаРаботы.
func (h *BudgetHandler) buildStageWorkKinds(ctx context.Context, contractID string, stages []models.Stage) map[string][]string {
	stagesMap := make(map[string]models.Stage)
	for _, s := range stages {
		stagesMap[s.GUID] = s
	}

	costs, _ := h.costSvc.GetAllCostsForContract(ctx, contractID)
	stageWorkKinds := make(map[string][]string)
	for _, c := range costs {
		if st, ok := stagesMap[c.StageGUID]; ok && st.StageNumber != "" {
			// Проверяем, нет ли уже такого вида работ (чтобы не было дублей)
			found := false
			for _, wk := range stageWorkKinds[st.StageNumber] {
				if wk == c.WorkKind {
					found = true
					break
				}
			}
			if !found {
				stageWorkKinds[st.StageNumber] = append(stageWorkKinds[st.StageNumber], c.WorkKind)
			}
		}
	}
	return stageWorkKinds
}

func (h *BudgetHandler) GetSummaryChart(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	selectedSnapshotID := r.URL.Query().Get("snapshot")

	var lines []models.BudgetPlanLine
	var err error

	if selectedSnapshotID != "" {
		_, lines, err = h.svc.GetPlanByID(r.Context(), selectedSnapshotID)
	} else {
		_, lines, err = h.svc.GetLatestPlan(r.Context(), contractID)
	}

	if err != nil {
		log.Printf("get summary chart lines: %v", err)
		http.Error(w, "Ошибка загрузки данных для графиков", http.StatusInternalServerError)
		return
	}

	expenseSums := make(map[string]float64)
	deptSums := make(map[string]float64)

	totalIncome := 0.0
	totalExpense := 0.0

	for _, l := range lines {
		totalIncome += l.Income
		totalExpense += l.Expense

		// В график затрат не включаем поступления от клиента (Платежи по договору)
		if l.OperationType != "Платежи по договору" && l.Expense > 0 {
			expenseSums[l.OperationType] += l.Expense
			deptSums[l.WorkKind] += l.Expense
		}
	}

	// Добавляем "Прибыль" в график затрат (как остаток)
	profit := totalIncome - totalExpense
	if profit != 0 {
		expenseSums["Прибыль"] = profit
	}

	// Формируем ответ
	resp := struct {
		ExpensesChart    ChartData `json:"expenses_chart"`
		DepartmentsChart ChartData `json:"departments_chart"`
	}{
		ExpensesChart:    ChartData{Labels: make([]string, 0, len(expenseSums)), Data: make([]float64, 0, len(expenseSums))},
		DepartmentsChart: ChartData{Labels: make([]string, 0, len(deptSums)), Data: make([]float64, 0, len(deptSums))},
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode summary chart: %v", err)
	}
}

type ChartData struct {
	Labels []string  `json:"labels"`
	Data   []float64 `json:"data"`
}

func (h *BudgetHandler) PlanFactIndex(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, err := h.contractSvc.GetByID(r.Context(), contractID)
	if err != nil || contract == nil {
		log.Printf("plan fact get contract %s: %v", contractID, err)
		http.Error(w, "Договор не найден", http.StatusNotFound)
		return
	}

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

	workKinds, err := h.stageSvc.WorkKinds(r.Context())
	if err != nil {
		log.Printf("get work kinds: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Группируем данные по видам работ для графика и итогов
	type WorkKindSummary struct {
		Code       string
		Name       string
		PlanProfit float64
		FactProfit float64
		Deviation  float64
	}

	// Справочник для перевода кода в имя
	wkNameMap := make(map[string]string)
	for _, wk := range workKinds {
		wkNameMap[wk.Code] = wk.Name
	}

	wkMap := make(map[string]*WorkKindSummary)
	for _, row := range summary {
		if _, ok := wkMap[row.WorkKind]; !ok {
			name := row.WorkKind
			if n, exists := wkNameMap[row.WorkKind]; exists {
				name = n
			}
			wkMap[row.WorkKind] = &WorkKindSummary{Code: row.WorkKind, Name: name}
		}
		wkMap[row.WorkKind].PlanProfit += row.PlanBalance
		wkMap[row.WorkKind].FactProfit += row.FactBalance
		wkMap[row.WorkKind].Deviation += row.Deviation
	}

	// Сортируем виды работ
	order := map[string]int{"ПИР": 1, "ППО": 2, "СМР": 3, "ПНР": 4, "Сборка": 5, "Продажа": 6}

	wkList := make([]*WorkKindSummary, 0, len(wkMap))
	for _, v := range wkMap {
		wkList = append(wkList, v)
	}

	sort.Slice(wkList, func(i, j int) bool {
		w1 := order[wkList[i].Code]
		if w1 == 0 {
			w1 = 99
		}
		w2 := order[wkList[j].Code]
		if w2 == 0 {
			w2 = 99
		}

		if w1 != w2 {
			return w1 < w2
		}
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
	if err := h.tmpl.ExecuteTemplate(w, "budget_compare.html", data); err != nil {
		log.Printf("render budget_compare: %v", err)
	}
}

func (h *BudgetHandler) FactDeleteRow(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	factID := r.FormValue("id")

	if factID != "" {
		// Вызываем метод удаления одной строки в сервисе (создадим его на след. шаге)
		if err := h.svc.DeleteFactLine(r.Context(), factID); err != nil {
			log.Printf("delete fact line: %v", err)
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *BudgetHandler) FactAppendFromPlan(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	if err := h.svc.AppendMissingFactLinesFromPlan(r.Context(), contractID); err != nil {
		log.Printf("append fact from plan: %v", err)
		http.Error(w, "Ошибка дозаполнения факта: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact?flash=Факт+дозаполнен+из+активного+плана", http.StatusSeeOther)
}

// FactAppendPreview отдает JSON со списком недостающих строк
func (h *BudgetHandler) FactAppendPreview(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	lines, err := h.svc.GetMissingPlanLines(r.Context(), contractID)
	if err != nil {
		log.Printf("get missing lines: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lines)
}

// FactAppendSelected принимает список ID и добавляет их в факт
func (h *BudgetHandler) FactAppendSelected(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")

	var payload struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.svc.AppendSelectedFactLines(r.Context(), contractID, payload.IDs); err != nil {
		log.Printf("ОШИБКА APPEND SELECTED: %v", err)
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
