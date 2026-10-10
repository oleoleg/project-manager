package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oleoleg/project-manager/internal/models"
)

var ErrCounterpartyNotFound = errors.New("counterparty not found")

type CounterpartyRepo struct {
	pool *pgxpool.Pool
}

func NewCounterpartyRepo(pool *pgxpool.Pool) *CounterpartyRepo {
	return &CounterpartyRepo{pool: pool}
}

const counterpartyColumns = `
	id, name, partner, short_name,
	address_actual, address_legal,
	url_workplace, url_servicework, url_salesprep,
	created_at, updated_at
`

func scanCounterparty(row pgx.Row, c *models.Counterparty) error {
	return row.Scan(
		&c.ID, &c.Name, &c.Partner, &c.ShortName,
		&c.AddressActual, &c.AddressLegal,
		&c.URLWorkplace, &c.URLServicework, &c.URLSalesprep,
		&c.CreatedAt, &c.UpdatedAt,
	)
}

func (r *CounterpartyRepo) GetByID(ctx context.Context, id string) (*models.Counterparty, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+counterpartyColumns+` FROM counterparties WHERE id = $1`, id)
	var c models.Counterparty
	if err := scanCounterparty(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCounterpartyNotFound
		}
		return nil, fmt.Errorf("get counterparty: %w", err)
	}
	return &c, nil
}

func (r *CounterpartyRepo) List(ctx context.Context, search string) ([]models.Counterparty, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if search == "" {
		rows, err = r.pool.Query(ctx, `
			SELECT `+counterpartyColumns+`
			FROM counterparties
			ORDER BY id
		`)
	} else {
		pattern := "%" + search + "%"
		rows, err = r.pool.Query(ctx, `
			SELECT `+counterpartyColumns+`
			FROM counterparties
			WHERE id ILIKE $1 OR name ILIKE $1 OR short_name ILIKE $1 OR partner ILIKE $1
			ORDER BY id
		`, pattern)
	}
	if err != nil {
		return nil, fmt.Errorf("list counterparties: %w", err)
	}
	defer rows.Close()

	var result []models.Counterparty
	for rows.Next() {
		var c models.Counterparty
		if err := scanCounterparty(rows, &c); err != nil {
			return nil, fmt.Errorf("scan counterparty: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *CounterpartyRepo) Create(ctx context.Context, c *models.Counterparty) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO counterparties
			(id, name, partner, short_name, address_actual, address_legal,
			 url_workplace, url_servicework, url_salesprep)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`,
		c.ID, c.Name, c.Partner, c.ShortName,
		c.AddressActual, c.AddressLegal,
		c.URLWorkplace, c.URLServicework, c.URLSalesprep,
	)
	if err != nil {
		return fmt.Errorf("create counterparty: %w", err)
	}
	return nil
}

func (r *CounterpartyRepo) Update(ctx context.Context, c *models.Counterparty) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE counterparties SET
			name = $2, partner = $3, short_name = $4,
			address_actual = $5, address_legal = $6,
			url_workplace = $7, url_servicework = $8, url_salesprep = $9,
			updated_at = NOW()
		WHERE id = $1
	`,
		c.ID, c.Name, c.Partner, c.ShortName,
		c.AddressActual, c.AddressLegal,
		c.URLWorkplace, c.URLServicework, c.URLSalesprep,
	)
	if err != nil {
		return fmt.Errorf("update counterparty: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCounterpartyNotFound
	}
	return nil
}

func (r *CounterpartyRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM counterparties WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete counterparty: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCounterpartyNotFound
	}
	return nil
}
