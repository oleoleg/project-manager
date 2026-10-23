package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type BudgetHandler struct {
	svc         *services.BudgetService
	contractSvc *services.ContractService
	stageSvc    *services.StageService
	tmpl        *template.Template
}

func NewBudgetHandler(svc *services.BudgetService, contractSvc *services.ContractService, stageSvc *services.StageService, tmpl *template.Template) *BudgetHandler {
	return &BudgetHandler{svc: svc, contractSvc: contractSvc, stageSvc: stageSvc, tmpl: tmpl}
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

	workKinds, _ := h.stageSvc.WorkKinds(r.Context())

	data := map[string]any{
		"Title":       "Бюджет договора",
		"User":        middleware.UserFromContext(r.Context()),
		"Contract":    contract,
		"Settings":    settings,
		"HasSettings": hasSettings,
		"Snapshots":   snapshots,
		"Snapshot":    activeSnapshot,
		"PlanLines":   lines,
		"Flash":       r.URL.Query().Get("flash"),
		"WorkKinds":   workKinds,
		"IsReadOnly":  isReadOnly,
		"EditMode":    editMode,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(w, "budget_index.html", data)
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

	// 2. Генерируем план
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

	if err := h.svc.ApproveSnapshot(r.Context(), snapshotID, u.ID); err != nil {
		log.Printf("approve snapshot: %v", err)
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
