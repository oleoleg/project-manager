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

	// Проверяем, включен ли режим редактирования (создание нового плана)
	editMode := r.URL.Query().Get("edit") == "1"

	// Получаем текущие (глобальные) настройки
	settings, hasSettings, _ := h.svc.GetSettings(r.Context(), contractID)

	// Получаем все снимки
	snapshots, _ := h.svc.GetAllSnapshots(r.Context(), contractID)

	// Определяем, какой снимок показывать
	selectedSnapshotID := r.URL.Query().Get("snapshot")
	var activeSnapshot *models.BudgetPlanSnapshot
	var lines []models.BudgetPlanLine

	if selectedSnapshotID != "" {
		activeSnapshot, lines, _ = h.svc.GetPlanByID(r.Context(), selectedSnapshotID)
	} else if len(snapshots) > 0 && !editMode {
		// Ищем активный снимок
		var targetID string
		for _, snap := range snapshots {
			if snap.IsActive {
				targetID = snap.ID
				break
			}
		}
		// Если активного нет, берем самый свежий
		if targetID == "" {
			targetID = snapshots[0].ID
		}
		activeSnapshot, lines, _ = h.svc.GetPlanByID(r.Context(), targetID)
	}

	// Если мы смотрим снимок, и у него сохранены настройки — берем их
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
		// Если смотрим снимок — берем замороженные затраты
		_ = json.Unmarshal(activeSnapshot.StageCostsSnapshot, &displayCosts)
	} else {
		// Если это новая версия (editMode) или снимков нет — берем актуальные из БД
		displayCosts, _ = h.costSvc.GetAllCostsForContract(r.Context(), contractID)
	}

	stages, _ := h.stageSvc.List(r.Context(), contractID)
	stagesMap := make(map[string]models.Stage)
	for _, s := range stages {
		stagesMap[s.GUID] = s
	}

	workKinds, _ := h.stageSvc.WorkKinds(r.Context())
	cp, _ := h.cpSvc.GetByID(r.Context(), contract.CounterpartyID)

	data := map[string]any{
		"Title":        "Бюджет договора",
		"User":         middleware.UserFromContext(r.Context()),
		"Contract":     contract,
		"Settings":     settings,
		"HasSettings":  hasSettings,
		"Snapshots":    snapshots,
		"Snapshot":     activeSnapshot,
		"PlanLines":    lines,
		"Flash":        r.URL.Query().Get("flash"),
		"WorkKinds":    workKinds,
		"IsReadOnly":   isReadOnly,
		"EditMode":     editMode,
		"Counterparty": cp,
		"Costs":        displayCosts,
		"StagesMap":    stagesMap,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_plan.html", data)
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

func (h *BudgetHandler) Generate(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	u := middleware.UserFromContext(r.Context())

	_ = r.ParseForm()
	note := r.FormValue("note")

	// 1. Сначала сохраняем настройки, которые пришли из формы (если мы в EditMode)
	// Мы всегда сохраняем, так как форма отправляет их сюда.
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

	// 2. СОХРАНЯЕМ ОТРЕДАКТИРОВАННЫЕ ЗАТРАТЫ В БД
	costs := parseCostsFromBudgetForm(r)

	// Группируем затраты по этапам, так как репозиторий ожидает сохранение по одному этапу
	costsByStage := make(map[string][]models.StageCost)
	for _, c := range costs {
		costsByStage[c.StageGUID] = append(costsByStage[c.StageGUID], c)
	}

	// Сохраняем
	for stageGUID, stageCosts := range costsByStage {
		if err := h.costSvc.ReplaceForStage(r.Context(), stageGUID, stageCosts); err != nil {
			log.Printf("save stage costs before generate: %v", err)
			http.Error(w, "Ошибка сохранения затрат: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 3. ГЕНЕРИРУЕМ ПЛАН (теперь он возьмет свежие затраты из БД)
	_, err := h.svc.GeneratePlan(r.Context(), contractID, u.ID, note)
	if err != nil {
		log.Printf("generate plan: %v", err)
		http.Error(w, "Ошибка: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget", http.StatusSeeOther)
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

	if err := h.svc.SetActiveSnapshot(r.Context(), contractID, snapshotID); err != nil {
		log.Printf("activate snapshot: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/budget", http.StatusSeeOther)
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

// ГРАФИКИ
type ChartData struct {
	Labels []string  `json:"labels"`
	Data   []float64 `json:"data"`
}

type SummaryResponse struct {
	ExpensesChart    ChartData `json:"expenses_chart"`
	DepartmentsChart ChartData `json:"departments_chart"`
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

	resp := SummaryResponse{
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

// FactIndex отображает страницу ФинФакта.
func (h *BudgetHandler) FactIndex(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	// Получаем строки факта (уже отсортированные по группам)
	lines, err := h.svc.GetFactLines(r.Context(), contractID)
	if err != nil {
		log.Printf("get fact lines: %v", err)
		lines = nil
	}

	// Группируем строки для шаблона: map[ТипОперации][]Строки
	groupedLines := make(map[string][]models.BudgetFactLine)
	var groupOrder []string

	// Задаем жесткий порядок групп
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

	// Сортируем порядок групп
	sort.Slice(groupOrder, func(i, j int) bool {
		return orderMap[groupOrder[i]] < orderMap[groupOrder[j]]
	})

	// ... (проверка предупреждений и активного плана оставляем как было)
	checkData, _ := h.svc.GetFactCheckData(r.Context(), contractID)
	var factPayments float64
	for _, l := range lines {
		// Сравниваем только подтвержденный факт с актами
		if l.OperationType == "Платежи по договору" && l.IsConfirmed {
			factPayments += l.FactIncome
		}
	}

	warnings := []string{}
	if math.Abs(factPayments-checkData["total_acts"]) > 1.0 && factPayments > 0 {
		warnings = append(warnings, fmt.Sprintf("Внимание: Сумма платежей по факту (%.2f ₽) не сходится с суммой закрытых актов в договоре (%.2f ₽).", factPayments, checkData["total_acts"]))
	}

	// Получаем все снимки и ищем среди них активный
	snapshots, _ := h.svc.GetAllSnapshots(r.Context(), contractID)
	hasActivePlan := false
	for _, snap := range snapshots {
		if snap.IsActive {
			hasActivePlan = true
			break
		}
	}
	workKinds, _ := h.stageSvc.WorkKinds(r.Context())
	cp, _ := h.cpSvc.GetByID(r.Context(), contract.CounterpartyID)

	data := map[string]any{
		"Title":         "ФинФакт договора",
		"User":          middleware.UserFromContext(r.Context()),
		"Contract":      contract,
		"Counterparty":  cp,
		"FactLines":     lines,
		"GroupedLines":  groupedLines, // Передаем сгруппированные строки
		"GroupOrder":    groupOrder,   // Передаем порядок групп
		"HasActivePlan": hasActivePlan,
		"WorkKinds":     workKinds,
		"Flash":         r.URL.Query().Get("flash"),
		"Warnings":      warnings,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_fact.html", data)
}

// FactGenerate создает факт на основе активного плана.
func (h *BudgetHandler) FactGenerate(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")

	if err := h.svc.GenerateFactFromPlan(r.Context(), contractID); err != nil {
		log.Printf("generate fact: %v", err)
		http.Error(w, "Ошибка генерации факта: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact?flash=Факт+успешно+сформирован", http.StatusSeeOther)
}

// FactDelete удаляет все фактические данные.
func (h *BudgetHandler) FactDelete(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")

	if err := h.svc.DeleteFact(r.Context(), contractID); err != nil {
		log.Printf("delete fact: %v", err)
		http.Error(w, "Ошибка удаления факта", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/contracts/"+contractID+"/budget/fact?flash=Факт+удален", http.StatusSeeOther)
}

// FactUpdate сохраняет изменения в конкретной строке факта.
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

	log.Printf("ПОЛУЧЕНО СТРОК: %d", len(payload.Lines))

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

	log.Println("УСПЕШНО СОХРАНЕНО")
	// Отдаем JSON, а не редирект, чтобы JS мог это обработать
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
	workKindCodes := r.Form["work_kind_code[]"]
	opTypes := r.Form["operation_type[]"]

	// Гарантируем, что все массивы одной длины (по количеству fact_id)
	n := len(ids)

	pad := func(arr []string) []string {
		if len(arr) >= n {
			return arr[:n]
		}
		out := make([]string, n)
		copy(out, arr)
		return out
	}

	// Чекбоксы — отдельная логика: они приходят только отмеченными
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
		ids, pad(names), pad(workKindCodes), pad(opTypes),
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

// NewVersion отображает страницу создания новой версии плана
func (h *BudgetHandler) NewVersion(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)

	// Получаем текущие настройки (или дефолтные)
	settings, _, _ := h.svc.GetSettings(r.Context(), contractID)

	// Получаем текущие затраты из БД (эталон)
	costs, _ := h.costSvc.GetAllCostsForContract(r.Context(), contractID)

	// Получаем этапы и справочники
	stages, _ := h.stageSvc.List(r.Context(), contractID)
	stagesMap := make(map[string]models.Stage)
	for _, s := range stages {
		stagesMap[s.GUID] = s
	}

	// Формируем полный список строк для таблицы (включая этапы без затрат)
	var displayCosts []models.StageCost

	// Сначала добавляем существующие затраты
	for _, c := range costs {
		displayCosts = append(displayCosts, c)
	}

	// Если у этапа нет ни одной строки затрат, добавляем фиктивную пустую строку для него
	for _, s := range stages {
		hasCosts := false
		for _, c := range costs {
			if c.StageGUID == s.GUID {
				hasCosts = true
				break
			}
		}
		if !hasCosts {
			displayCosts = append(displayCosts, models.StageCost{
				StageGUID:    s.GUID,
				WorkKindCode: s.WorkKindCode, // Берем вид работы самого этапа
			})
		}
	}

	workKinds, _ := h.stageSvc.WorkKinds(r.Context())
	cp, _ := h.cpSvc.GetByID(r.Context(), contract.CounterpartyID)

	data := map[string]any{
		"Title":        "Новая версия бюджета",
		"User":         middleware.UserFromContext(r.Context()),
		"Contract":     contract,
		"Counterparty": cp,
		"Settings":     settings,
		"Costs":        displayCosts,
		"StagesMap":    stagesMap,
		"WorkKinds":    workKinds,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_new.html", data)
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
		// Убираем пробелы и меняем запятую на точку
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
