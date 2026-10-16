package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oleoleg/project-manager/internal/models"
)

var ErrActNotFound = errors.New("act not found")

type ActRepo struct {
	pool *pgxpool.Pool
}

func NewActRepo(pool *pgxpool.Pool) *ActRepo {
	return &ActRepo{pool: pool}
}

const actColumns = `
	guid, kind, kind_custom, name, amount, sign_date,
	file_path, note, distribution_expanded, created_at, updated_at
`

func scanAct(row pgx.Row, a *models.Act) error {
	return row.Scan(
		&a.GUID, &a.Kind, &a.KindCustom, &a.Name, &a.Amount, &a.SignDate,
		&a.FilePath, &a.Note, &a.DistributionExpanded, &a.CreatedAt, &a.UpdatedAt,
	)
}

// GetByID возвращает акт по GUID.
func (r *ActRepo) GetByID(ctx context.Context, guid string) (*models.Act, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+actColumns+` FROM acts WHERE guid = $1`, guid)
	var a models.Act
	if err := scanAct(row, &a); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrActNotFound
		}
		return nil, fmt.Errorf("get act: %w", err)
	}
	return &a, nil
}

// ListByStage возвращает главные акты, привязанные к этапу.
func (r *ActRepo) ListByStage(ctx context.Context, stageGUID string) ([]models.Act, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+actColumns+`
		FROM acts
		WHERE guid IN (SELECT act_guid FROM act_stages WHERE stage_guid = $1)
		ORDER BY sign_date NULLS LAST, created_at
	`, stageGUID)
	if err != nil {
		return nil, fmt.Errorf("list acts by stage: %w", err)
	}
	defer rows.Close()

	var res []models.Act
	for rows.Next() {
		var a models.Act
		if err := scanAct(rows, &a); err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, rows.Err()
}

// Create создаёт акт.
func (r *ActRepo) Create(ctx context.Context, a *models.Act) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO acts (kind, kind_custom, name, amount, sign_date, file_path, note, distribution_expanded)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING guid, created_at, updated_at
	`,
		a.Kind, a.KindCustom, a.Name, a.Amount, a.SignDate, a.FilePath, a.Note, a.DistributionExpanded,
	).Scan(&a.GUID, &a.CreatedAt, &a.UpdatedAt)
}

// Update обновляет акт.
func (r *ActRepo) Update(ctx context.Context, a *models.Act) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE acts SET
			kind = $2, kind_custom = $3, name = $4, amount = $5,
			sign_date = $6, file_path = $7, note = $8,
			distribution_expanded = $9,
			updated_at = NOW()
		WHERE guid = $1
	`,
		a.GUID, a.Kind, a.KindCustom, a.Name, a.Amount,
		a.SignDate, a.FilePath, a.Note, a.DistributionExpanded,
	)
	if err != nil {
		return fmt.Errorf("update act: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActNotFound
	}
	return nil
}

// Delete удаляет акт и все его связи (каскадно).
func (r *ActRepo) Delete(ctx context.Context, guid string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM acts WHERE guid = $1`, guid)
	if err != nil {
		return fmt.Errorf("delete act: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActNotFound
	}
	return nil
}

// --- связи с этапами ---

// LinkStage привязывает акт к этапу с указанной суммой.
func (r *ActRepo) LinkStage(ctx context.Context, link models.ActStageLink) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO act_stages (act_guid, stage_guid, amount)
		VALUES ($1, $2, $3)
		ON CONFLICT (act_guid, stage_guid) DO UPDATE SET amount = EXCLUDED.amount
	`, link.ActGUID, link.StageGUID, link.Amount)
	if err != nil {
		return fmt.Errorf("link act to stage: %w", err)
	}
	return nil
}

// UnlinkStage отвязывает акт от этапа.
func (r *ActRepo) UnlinkStage(ctx context.Context, actGUID, stageGUID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM act_stages WHERE act_guid = $1 AND stage_guid = $2
	`, actGUID, stageGUID)
	if err != nil {
		return fmt.Errorf("unlink act from stage: %w", err)
	}
	return nil
}

// SumByStage — сумма распределённых по этапу актов (без НДС).
func (r *ActRepo) SumByStage(ctx context.Context, stageGUID string) (float64, error) {
	var sum float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM act_stages WHERE stage_guid = $1
	`, stageGUID).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum act_stages: %w", err)
	}
	return sum, nil
}

// SumByAct — сколько из акта уже распределено по этапам.
func (r *ActRepo) SumByAct(ctx context.Context, actGUID string) (float64, error) {
	var sum float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM act_stages WHERE act_guid = $1
	`, actGUID).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum act_stages by act: %w", err)
	}
	return sum, nil
}

// StagesLinkedToAct — список связей акта с этапами.
func (r *ActRepo) StagesLinkedToAct(ctx context.Context, actGUID string) ([]models.ActStageLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT act_guid, stage_guid, amount
		FROM act_stages WHERE act_guid = $1
	`, actGUID)
	if err != nil {
		return nil, fmt.Errorf("list act_stages: %w", err)
	}
	defer rows.Close()

	var res []models.ActStageLink
	for rows.Next() {
		var l models.ActStageLink
		if err := rows.Scan(&l.ActGUID, &l.StageGUID, &l.Amount); err != nil {
			return nil, err
		}
		res = append(res, l)
	}
	return res, rows.Err()
}

// StageGUIDByAct — GUID первого этапа, к которому привязан акт.
func (r *ActRepo) StageGUIDByAct(ctx context.Context, actGUID string) (string, error) {
	var stageGUID string
	err := r.pool.QueryRow(ctx, `
		SELECT stage_guid FROM act_stages WHERE act_guid = $1 LIMIT 1
	`, actGUID).Scan(&stageGUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("stage guid by act: %w", err)
	}
	return stageGUID, nil
}

// CountByContract — map stageGUID → количество актов.
func (r *ActRepo) CountByContract(ctx context.Context, contractID string) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ast.stage_guid, COUNT(DISTINCT ast.act_guid)::int
		FROM act_stages ast
		JOIN stages s ON s.guid = ast.stage_guid
		WHERE s.contract_id = $1
		GROUP BY ast.stage_guid
	`, contractID)
	if err != nil {
		return nil, fmt.Errorf("count acts by contract: %w", err)
	}
	defer rows.Close()

	res := make(map[string]int)
	for rows.Next() {
		var guid string
		var n int
		if err := rows.Scan(&guid, &n); err != nil {
			return nil, err
		}
		res[guid] = n
	}
	return res, rows.Err()
}

// --- приложения ---

const attachmentColumns = `
	guid, act_guid, kind, kind_custom, name, file_path, note, created_at, updated_at
`

func scanAttachment(row pgx.Row, a *models.ActAttachment) error {
	return row.Scan(
		&a.GUID, &a.ActGUID, &a.Kind, &a.KindCustom, &a.Name,
		&a.FilePath, &a.Note, &a.CreatedAt, &a.UpdatedAt,
	)
}

// ListAttachmentsByAct — список приложений главного акта.
func (r *ActRepo) ListAttachmentsByAct(ctx context.Context, actGUID string) ([]models.ActAttachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+attachmentColumns+`
		FROM act_attachments WHERE act_guid = $1
		ORDER BY created_at
	`, actGUID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	var res []models.ActAttachment
	for rows.Next() {
		var a models.ActAttachment
		if err := scanAttachment(rows, &a); err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, rows.Err()
}

// CreateAttachment создаёт приложение.
func (r *ActRepo) CreateAttachment(ctx context.Context, a *models.ActAttachment) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO act_attachments (act_guid, kind, kind_custom, name, file_path, note)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING guid, created_at, updated_at
	`,
		a.ActGUID, a.Kind, a.KindCustom, a.Name, a.FilePath, a.Note,
	).Scan(&a.GUID, &a.CreatedAt, &a.UpdatedAt)
}

// DeleteAttachment удаляет приложение.
func (r *ActRepo) DeleteAttachment(ctx context.Context, guid string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM act_attachments WHERE guid = $1`, guid)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	return nil
}

// GetAttachmentByID возвращает приложение по GUID.
func (r *ActRepo) GetAttachmentByID(ctx context.Context, guid string) (*models.ActAttachment, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+attachmentColumns+` FROM act_attachments WHERE guid = $1`, guid)
	var a models.ActAttachment
	if err := scanAttachment(row, &a); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrActNotFound
		}
		return nil, err
	}
	return &a, nil
}

// заглушка импорта time (используется в других методах)
var _ = time.Now

// AmountByActAndStage возвращает, сколько из акта идёт на конкретный этап.
// Если связи нет — 0.
func (r *ActRepo) AmountByActAndStage(ctx context.Context, actGUID, stageGUID string) (float64, error) {
	var amount float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(amount, 0)
		FROM act_stages
		WHERE act_guid = $1 AND stage_guid = $2
	`, actGUID, stageGUID).Scan(&amount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("amount by act and stage: %w", err)
	}
	return amount, nil
}

// UpdateAttachment обновляет приложение.
func (r *ActRepo) UpdateAttachment(ctx context.Context, a *models.ActAttachment) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE act_attachments SET
			kind = $2, kind_custom = $3, name = $4,
			file_path = $5, note = $6, updated_at = NOW()
		WHERE guid = $1
	`,
		a.GUID, a.Kind, a.KindCustom, a.Name, a.FilePath, a.Note,
	)
	if err != nil {
		return fmt.Errorf("update attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActNotFound
	}
	return nil
}
