package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oleoleg/project-manager/internal/models"
)

var ErrDocumentNotFound = errors.New("document not found")
var ErrVersionNotFound = errors.New("document version not found")

type DocumentRepo struct {
	pool *pgxpool.Pool
}

func NewDocumentRepo(pool *pgxpool.Pool) *DocumentRepo {
	return &DocumentRepo{pool: pool}
}

// ListByContract возвращает все документы договора вместе с их версиями и историей согласования.
func (r *DocumentRepo) ListByContract(ctx context.Context, contractID string) ([]models.ContractDocument, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contract_id, doc_type, number, doc_date, title, current_stage, current_form, holder_info, created_at, updated_at
		FROM contract_documents
		WHERE contract_id = $1
		ORDER BY 
			CASE doc_type WHEN 'main' THEN 1 WHEN 'addendum' THEN 2 ELSE 3 END,
			created_at ASC
	`, contractID)
	if err != nil {
		return nil, fmt.Errorf("list contract documents: %w", err)
	}
	defer rows.Close()

	var docs []models.ContractDocument
	for rows.Next() {
		var d models.ContractDocument
		if err := rows.Scan(
			&d.ID, &d.ContractID, &d.DocType, &d.Number, &d.DocDate, &d.Title,
			&d.CurrentStage, &d.CurrentForm, &d.HolderInfo, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan contract document: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Подгружаем версии и историю для каждого документа
	for i := range docs {
		versions, err := r.listVersions(ctx, docs[i].ID)
		if err != nil {
			return nil, err
		}
		docs[i].Versions = versions

		history, err := r.listHistory(ctx, docs[i].ID)
		if err != nil {
			return nil, err
		}
		docs[i].History = history
	}

	return docs, nil
}

func (r *DocumentRepo) listVersions(ctx context.Context, docID int64) ([]models.DocumentVersion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.document_id, v.version_num, v.rel_path, v.attachments_rel_path,
		       v.file_name, v.file_ext, v.form_type, v.comment, v.uploaded_by,
		       COALESCE(u.full_name, ''), v.created_at
		FROM document_versions v
		LEFT JOIN users u ON u.id = v.uploaded_by
		WHERE v.document_id = $1
		ORDER BY v.version_num DESC
	`, docID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var res []models.DocumentVersion
	for rows.Next() {
		var v models.DocumentVersion
		if err := rows.Scan(
			&v.ID, &v.DocumentID, &v.VersionNum, &v.RelPath, &v.AttachmentsRelPath,
			&v.FileName, &v.FileExt, &v.FormType, &v.Comment, &v.UploadedBy,
			&v.UploaderName, &v.CreatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, rows.Err()
}

func (r *DocumentRepo) listHistory(ctx context.Context, docID int64) ([]models.DocumentHistory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT h.id, h.document_id, h.version_id, COALESCE(v.version_num, 0),
		       h.stage, h.form_type, h.recipient_info, h.note,
		       h.created_by, COALESCE(u.full_name, ''), h.created_at
		FROM document_history h
		LEFT JOIN document_versions v ON v.id = h.version_id
		LEFT JOIN users u ON u.id = h.created_by
		WHERE h.document_id = $1
		ORDER BY h.created_at DESC
	`, docID)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var res []models.DocumentHistory
	for rows.Next() {
		var h models.DocumentHistory
		if err := rows.Scan(
			&h.ID, &h.DocumentID, &h.VersionID, &h.VersionNum,
			&h.Stage, &h.FormType, &h.RecipientInfo, &h.Note,
			&h.CreatedBy, &h.AuthorName, &h.CreatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, h)
	}
	return res, rows.Err()
}

func (r *DocumentRepo) GetDocumentByID(ctx context.Context, docID int64) (*models.ContractDocument, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, contract_id, doc_type, number, doc_date, title, current_stage, current_form, holder_info, created_at, updated_at
		FROM contract_documents WHERE id = $1
	`, docID)
	var d models.ContractDocument
	if err := row.Scan(
		&d.ID, &d.ContractID, &d.DocType, &d.Number, &d.DocDate, &d.Title,
		&d.CurrentStage, &d.CurrentForm, &d.HolderInfo, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *DocumentRepo) GetVersionByID(ctx context.Context, verID int64) (*models.DocumentVersion, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, document_id, version_num, rel_path, attachments_rel_path,
		       file_name, file_ext, form_type, comment, uploaded_by, created_at
		FROM document_versions WHERE id = $1
	`, verID)
	var v models.DocumentVersion
	if err := row.Scan(
		&v.ID, &v.DocumentID, &v.VersionNum, &v.RelPath, &v.AttachmentsRelPath,
		&v.FileName, &v.FileExt, &v.FormType, &v.Comment, &v.UploadedBy, &v.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}
	return &v, nil
}

// CreateDocument создаёт логическую карточку документа (Договор / ДС / Приложение).
func (r *DocumentRepo) CreateDocument(ctx context.Context, d *models.ContractDocument) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO contract_documents (contract_id, doc_type, number, doc_date, title, current_stage, current_form, holder_info)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, d.ContractID, d.DocType, d.Number, d.DocDate, d.Title, d.CurrentStage, d.CurrentForm, d.HolderInfo).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

// NextVersionNumber возвращает следующий номер версии (1, 2, 3...) для документа.
func (r *DocumentRepo) NextVersionNumber(ctx context.Context, docID int64) (int, error) {
	var maxVer int
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version_num), 0) FROM document_versions WHERE document_id = $1`, docID).Scan(&maxVer)
	return maxVer + 1, err
}

// AddVersion сохраняет запись о версии файла и обновляет текущий вид документа в родительской таблице.
func (r *DocumentRepo) AddVersion(ctx context.Context, v *models.DocumentVersion) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO document_versions (
			document_id, version_num, rel_path, attachments_rel_path,
			file_name, file_ext, form_type, comment, uploaded_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`, v.DocumentID, v.VersionNum, v.RelPath, v.AttachmentsRelPath, v.FileName, v.FileExt, v.FormType, v.Comment, v.UploadedBy).
		Scan(&v.ID, &v.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert document version: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE contract_documents
		SET current_form = $2, updated_at = NOW()
		WHERE id = $1
	`, v.DocumentID, v.FormType)
	if err != nil {
		return fmt.Errorf("update document form: %w", err)
	}

	return tx.Commit(ctx)
}

// AddHistory добавляет запись в журнал согласования и обновляет текущий статус/местонахождение документа.
func (r *DocumentRepo) AddHistory(ctx context.Context, h *models.DocumentHistory) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO document_history (document_id, version_id, stage, form_type, recipient_info, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, h.DocumentID, h.VersionID, h.Stage, h.FormType, h.RecipientInfo, h.Note, h.CreatedBy).
		Scan(&h.ID, &h.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert document history: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE contract_documents
		SET current_stage = $2, current_form = $3, holder_info = $4, updated_at = NOW()
		WHERE id = $1
	`, h.DocumentID, h.Stage, h.FormType, h.RecipientInfo)
	if err != nil {
		return fmt.Errorf("update document status: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *DocumentRepo) DeleteDocument(ctx context.Context, docID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM contract_documents WHERE id = $1`, docID)
	return err
}

// UpdateDocument обновляет метаданные карточки документа (тип, номер, название, местонахождение).
func (r *DocumentRepo) UpdateDocument(ctx context.Context, d *models.ContractDocument) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE contract_documents
		SET number = $2, doc_date = $3, title = $4, holder_info = $5, updated_at = NOW()
		WHERE id = $1
	`, d.ID, d.Number, d.DocDate, d.Title, d.HolderInfo)
	if err != nil {
		return fmt.Errorf("update contract document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

// GetDocumentFull возвращает один документ вместе с его версиями и историей.
func (r *DocumentRepo) GetDocumentFull(ctx context.Context, docID int64) (*models.ContractDocument, error) {
	d, err := r.GetDocumentByID(ctx, docID)
	if err != nil {
		return nil, err
	}
	d.Versions, err = r.listVersions(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	d.History, err = r.listHistory(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// UpdateVersionPath обновляет путь и имя файла версии (при перемещении в другую папку этапа).
func (r *DocumentRepo) UpdateVersionPath(ctx context.Context, verID int64, newRelPath, newFileName, newAttRelPath string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE document_versions
		SET rel_path = $2, file_name = $3, attachments_rel_path = $4
		WHERE id = $1
	`, verID, newRelPath, newFileName, newAttRelPath)
	return err
}

// DeleteVersion удаляет запись о версии из БД (сам файл на диске не трогается).
func (r *DocumentRepo) DeleteVersion(ctx context.Context, verID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM document_versions WHERE id = $1`, verID)
	return err
}

func (r *DocumentRepo) HasMainDocument(ctx context.Context, contractID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM contract_documents WHERE contract_id = $1 AND doc_type = 'main')
	`, contractID).Scan(&exists)
	return exists, err
}
