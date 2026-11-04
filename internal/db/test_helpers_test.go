package db_test

import (
	"context"
	"testing"
)

// clearTestDB полностью очищает все таблицы перед тестом.
func clearTestDB(t *testing.T) {
	ctx := context.Background()
	_, err := testPool.Exec(ctx, `
		TRUNCATE TABLE 
			budget_fact_lines, 
			budget_plan_lines, 
			budget_plan_snapshots, 
			budget_settings, 
			stage_costs, 
			stages, 
			contract_cards, 
			contracts, 
			counterparties, 
			users 
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("не удалось очистить БД: %v", err)
	}
}
