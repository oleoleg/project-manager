package services_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	_ = godotenv.Load("../../.env.test")

	connString := os.Getenv("TEST_DATABASE_URL")
	if connString == "" {
		connString = "postgres://postgres:postgres@localhost:5432/project_manager_test"
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		fmt.Printf("Не удалось подключиться к тестовой БД: %v\n", err)
		os.Exit(1)
	}
	defer testPool.Close()

	code := m.Run()
	os.Exit(code)
}

func TestBudgetService_GeneratePlan(t *testing.T) {
	ctx := context.Background()
	budgetRepo := db.NewBudgetRepo(testPool)
	budgetSvc := services.NewBudgetService(budgetRepo)
	stageRepo := db.NewStageRepo(testPool)
	costRepo := db.NewStageCostRepo(testPool)

	// 1. Подготовка БД (очистка и базовые данные)
	clearTestDB(t)

	contractID := "9999-01"
	cpID := "9999"
	userID := int64(1)

	_, _ = testPool.Exec(ctx, `INSERT INTO counterparties (id, name) VALUES ($1, 'Cp')`, cpID)
	_, _ = testPool.Exec(ctx, `INSERT INTO users (id, username, full_name, password_hash, role_code, is_active) VALUES ($1, 'u', 'U', 'h', 'admin', true)`, userID)
	_, _ = testPool.Exec(ctx, `INSERT INTO contracts (id, contract_number, contract_date, status_code, counterparty_id, contract_type) VALUES ($1, 'TEST', NOW(), 1, $2, 'contract')`, contractID, cpID)

	// 2. Создаем этап
	closeDate := time.Now().AddDate(0, 1, 0) // Закрытие через месяц
	stage := &models.Stage{
		ContractID:     contractID,
		StageNumber:    "1",
		Name:           "Этап 1",
		Cost:           100000, // Стоимость 100 000
		AdvancePercent: 20,     // Аванс 20%
		VATPercent:     20,
		StatusCode:     1,
		WorkKind:       "ПИР", // Простой этап (не комплексный)
		CloseDatePlan:  &closeDate,
	}
	err := stageRepo.Create(ctx, stage)
	if err != nil {
		t.Fatalf("ошибка создания этапа: %v", err)
	}

	// 3. Создаем затраты для этапа
	cost := &models.StageCost{
		StageGUID:   stage.GUID,
		WorkKind:    "ПИР",
		Cost:        100000, // Стоимость работ
		FOTPlan:     20000,  // ФОТп 20 000
		Subcontract: 10000,  // Субподряд 10 000
	}
	err = costRepo.ReplaceForStage(ctx, stage.GUID, []models.StageCost{*cost})
	if err != nil {
		t.Fatalf("ошибка создания затрат: %v", err)
	}

	// 4. Сохраняем настройки бюджета (Коэф. накладных = 1.5, т.е. 150%)
	settings := models.BudgetSettings{
		ContractID:          contractID,
		ProjectStartDate:    time.Now(),
		OverheadRate:        1.5,
		PaymentDelayDays:    10,
		EquipPaymentPercent: 0,
	}
	_ = budgetSvc.SaveSettings(ctx, settings)

	// 5. ВЫЗЫВАЕМ ТЕСТИРУЕМЫЙ МЕТОД
	_, err = budgetSvc.GeneratePlan(ctx, contractID, userID, "Тестовый план")
	if err != nil {
		t.Fatalf("ошибка генерации плана: %v", err)
	}

	// 6. ПРОВЕРЯЕМ РЕЗУЛЬТАТЫ
	// Получаем сгенерированные строки плана
	lines, err := budgetRepo.GetPlanLines(ctx, contractID)
	if err != nil {
		t.Fatalf("ошибка получения плана: %v", err)
	}

	// Проверяем, что строки вообще есть
	if len(lines) == 0 {
		t.Fatal("план пуст, строки не сгенерировались")
	}

	// Проверяем конкретные суммы
	var foundAdvance, foundPayment, foundSalary, foundOverhead, foundSubcontract bool

	for _, l := range lines {
		switch l.OperationType {
		case db.OpTypePayment:
			if strings.Contains(l.Name, "Аванс") {
				foundAdvance = true
				// Аванс = 20% от 100 000 = 20 000
				if l.Income != 20000 {
					t.Errorf("неверная сумма аванса: ожидалось 20000, получено %.2f", l.Income)
				}
			} else if strings.Contains(l.Name, "Оплата") {
				foundPayment = true
				// Остаток = 80 000
				if l.Income != 80000 {
					t.Errorf("неверная сумма оплаты: ожидалось 80000, получено %.2f", l.Income)
				}
			}
		case db.OpTypeSalary:
			foundSalary = true
			// Зарплата = ФОТп = 20 000
			if l.Expense != 20000 {
				t.Errorf("неверная сумма ЗП: ожидалось 20000, получено %.2f", l.Expense)
			}
		case db.OpTypeOverhead:
			foundOverhead = true
			// Накладные = ФОТп * 1.5 = 20 000 * 1.5 = 30 000
			if l.Expense != 30000 {
				t.Errorf("неверная сумма Накладных: ожидалось 30000, получено %.2f", l.Expense)
			}
		case db.OpTypeSubcontract:
			foundSubcontract = true
			// Субподряд = 10 000
			if l.Expense != 10000 {
				t.Errorf("неверная сумма Субподряда: ожидалось 10000, получено %.2f", l.Expense)
			}
		}
	}

	// Проверяем, что все нужные строки сгенерировались
	if !foundAdvance {
		t.Error("не найдена строка Аванса")
	}
	if !foundPayment {
		t.Error("не найдена строка Оплаты")
	}
	if !foundSalary {
		t.Error("не найдена строка Зарплаты")
	}
	if !foundOverhead {
		t.Error("не найдена строка Накладных")
	}
	if !foundSubcontract {
		t.Error("не найдена строка Субподряда")
	}
}

// Вспомогательная функция для копирования (чтобы не тянуть из другого пакета)
func clearTestDB(t *testing.T) {
	ctx := context.Background()
	_, err := testPool.Exec(ctx, `
		TRUNCATE TABLE 
			budget_fact_lines, budget_plan_lines, budget_plan_snapshots, budget_settings, 
			stage_costs, stages, contract_cards, contracts, counterparties, users 
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("не удалось очистить БД: %v", err)
	}
}
