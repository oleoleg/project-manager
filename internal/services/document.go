package services

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var invalidFileChars = regexp.MustCompile(`[<>:"/\\|?*]`)

// BuildDocRelPath формирует относительный путь:
// - Не подписан: 2) Договор\Основной_договор\На_доработке\v1_Договор_№_123_от_2026-11-22_draft.docx
// - Подписан:    2) Договор\Основной_договор\Договор_№_123_от_2026-11-22_signed.pdf
func BuildDocRelPath(
	c *models.Contract,
	doc *models.ContractDocument,
	stage string,
	formType string,
	ext string,
	versionNum int,
) string {
	// 1. Папка типа документа
	docFolder := "Основной_договор"
	cleanDocNum := sanitizeFilename(doc.Number)
	switch doc.DocType {
	case "addendum":
		if cleanDocNum != "" {
			docFolder = "ДС_" + cleanDocNum
		} else {
			docFolder = "Доп_соглашения"
		}
	case "appendix":
		if cleanDocNum != "" {
			docFolder = "Приложение_" + cleanDocNum
		} else {
			docFolder = "Приложения"
		}
	}

	// 2. Базовое имя сущности (Договор / ДС_1 / Приложение_2)
	docLabel := "Договор"
	if doc.DocType == "addendum" {
		docLabel = "ДС"
		if cleanDocNum != "" {
			docLabel += "_" + cleanDocNum
		}
	} else if doc.DocType == "appendix" {
		docLabel = "Приложение"
		if cleanDocNum != "" {
			docLabel += "_" + cleanDocNum
		}
	}

	// 3. Номер и дата договора (если присвоены)
	var nameParts []string
	if stage != "signed" {
		nameParts = append(nameParts, fmt.Sprintf("v%d", versionNum))
	}
	nameParts = append(nameParts, docLabel)

	if c != nil {
		if num := sanitizeFilename(c.ContractNumber); num != "" {
			nameParts = append(nameParts, "№_"+num)
		}
		if c.ContractDate != nil {
			nameParts = append(nameParts, "от_"+c.ContractDate.Format("2006-01-02"))
		}
	}

	// 4. Суффикс статуса/вида
	if stage == "signed" {
		nameParts = append(nameParts, "signed")
	} else {
		if formType == "" {
			formType = "draft"
		}
		nameParts = append(nameParts, sanitizeFilename(formType))
	}

	cleanExt := strings.TrimPrefix(ext, ".")
	fileName := strings.Join(nameParts, "_") + "." + cleanExt

	// 5. Сборка итогового пути
	if stage == "signed" {
		return filepath.Join("2) Договор", docFolder, fileName)
	}
	return filepath.Join("2) Договор", docFolder, StageFolderName(stage), fileName)
}

// SaveFileToUNC записывает файл по UNC-пути и ставит атрибут Read-Only (0444).
func SaveFileToUNC(projectURL, relPath string, src io.Reader) (string, error) {
	if projectURL == "" {
		return "", fmt.Errorf("у договора не указан путь к папке проекта")
	}

	fullPath := filepath.Join(projectURL, relPath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("не удалось создать директорию на сервере (%s): %w", dir, err)
	}

	// На случай если файл уже существовал с атрибутом Read-Only — снимаем защиту перед перезаписью
	_ = os.Chmod(fullPath, 0666)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("не удалось создать файл на сервере (%s): %w", fullPath, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("ошибка записи файла: %w", err)
	}
	if err := dst.Close(); err != nil {
		return "", err
	}

	// Вешаем атрибут "Только для чтения" (в Windows os.Chmod(0444) ставит FILE_ATTRIBUTE_READONLY)
	_ = os.Chmod(fullPath, 0444)

	return fullPath, nil
}

// CopyExistingFileToUNC копирует уже лежащий на сервере файл в архив версий с Read-Only.
func CopyExistingFileToUNC(projectURL, sourceUNCPath, relPath string) (string, error) {
	sourceUNCPath = strings.TrimSpace(sourceUNCPath)
	if !strings.HasPrefix(sourceUNCPath, `\\`) {
		return "", fmt.Errorf("выберите файл с сетевого сервера (путь должен начинаться с \\\\)")
	}

	srcFile, err := os.Open(sourceUNCPath)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть исходный файл на сервере: %w", err)
	}
	defer srcFile.Close()

	return SaveFileToUNC(projectURL, relPath, srcFile)
}

// MoveFileOnUNC перемещает файл (например, из подпапки согласования в корень при подписании), сохраняя Read-Only.
func MoveFileOnUNC(projectURL, oldRelPath, newRelPath string) error {
	oldFull := filepath.Join(projectURL, oldRelPath)
	newFull := filepath.Join(projectURL, newRelPath)

	if oldFull == newFull {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(newFull), 0755); err != nil {
		return err
	}

	// В Windows переименование Read-Only файла может требовать временного снятия атрибута
	_ = os.Chmod(oldFull, 0666)
	_ = os.Chmod(newFull, 0666)

	if err := os.Rename(oldFull, newFull); err != nil {
		// Если перемещение между разными томами — копируем и удаляем старый
		f, openErr := os.Open(oldFull)
		if openErr != nil {
			_ = os.Chmod(oldFull, 0444)
			return err
		}
		_, saveErr := SaveFileToUNC(projectURL, newRelPath, f)
		f.Close()
		if saveErr != nil {
			_ = os.Chmod(oldFull, 0444)
			return saveErr
		}
		_ = os.Remove(oldFull)
		return nil
	}

	_ = os.Chmod(newFull, 0444)
	return nil
}

// GenerateDocxFromTemplate формирует .docx из шаблона и вешает Read-Only.
func GenerateDocxFromTemplate(templatePath, destFullPath string, replacements map[string]string) error {
	r, err := zip.OpenReader(templatePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия шаблона docx: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(filepath.Dir(destFullPath), 0755); err != nil {
		return fmt.Errorf("ошибка создания папки: %w", err)
	}

	_ = os.Chmod(destFullPath, 0666)
	outFile, err := os.Create(destFullPath)
	if err != nil {
		return fmt.Errorf("ошибка создания выходного docx: %w", err)
	}

	w := zip.NewWriter(outFile)
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			w.Close()
			outFile.Close()
			return err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			w.Close()
			outFile.Close()
			return err
		}

		if strings.HasPrefix(f.Name, "word/") && strings.HasSuffix(f.Name, ".xml") {
			text := string(content)
			for placeholder, val := range replacements {
				text = strings.ReplaceAll(text, placeholder, xmlEscape(val))
			}
			content = []byte(text)
		}

		fw, err := w.Create(f.Name)
		if err != nil {
			w.Close()
			outFile.Close()
			return err
		}
		if _, err := io.Copy(fw, bytes.NewReader(content)); err != nil {
			w.Close()
			outFile.Close()
			return err
		}
	}

	if err := w.Close(); err != nil {
		outFile.Close()
		return err
	}
	if err := outFile.Close(); err != nil {
		return err
	}

	_ = os.Chmod(destFullPath, 0444)
	return nil
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "_")
	return invalidFileChars.ReplaceAllString(s, "")
}

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

var (
	ErrDocTitleRequired      = errors.New("укажите наименование документа")
	ErrDocProjectURLRequired = errors.New("у договора не задан путь к папке проекта на сервере")
	ErrDocInvalidExt         = errors.New("допускаются только файлы PDF, DOC и DOCX")
	ErrDocTemplateRequired   = errors.New("выберите шаблон документа")
)

type DocumentService struct {
	repo         *db.DocumentRepo
	contractRepo *db.ContractRepo
	cpRepo       *db.CounterpartyRepo
	templatesDir string
}

func NewDocumentService(
	repo *db.DocumentRepo,
	contractRepo *db.ContractRepo,
	cpRepo *db.CounterpartyRepo,
	templatesDir string,
) *DocumentService {
	return &DocumentService{
		repo:         repo,
		contractRepo: contractRepo,
		cpRepo:       cpRepo,
		templatesDir: templatesDir,
	}
}

func (s *DocumentService) ListByContract(ctx context.Context, contractID string) ([]models.ContractDocument, error) {
	return s.repo.ListByContract(ctx, contractID)
}

func (s *DocumentService) GetVersionByID(ctx context.Context, verID int64) (*models.DocumentVersion, error) {
	return s.repo.GetVersionByID(ctx, verID)
}

func (s *DocumentService) CreateDocument(ctx context.Context, d *models.ContractDocument) error {
	d.Title = strings.TrimSpace(d.Title)
	d.Number = strings.TrimSpace(d.Number)
	d.HolderInfo = strings.TrimSpace(d.HolderInfo)

	if d.Title == "" {
		return ErrDocTitleRequired
	}
	if d.DocType == "" {
		d.DocType = "main"
	}
	if d.CurrentStage == "" {
		d.CurrentStage = "draft"
	}
	if d.CurrentForm == "" {
		d.CurrentForm = "draft"
	}

	return s.repo.CreateDocument(ctx, d)
}

// UploadVersion сохраняет файл на UNC-сервер и регистрирует новую версию в БД.
func (s *DocumentService) UploadVersion(
	ctx context.Context,
	contractID string,
	docID int64,
	origFilename string,
	fileReader io.Reader,
	formType string,
	comment string,
	userID *int64,
) (int, error) {
	c, err := s.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return 0, err
	}
	if c.ProjectURL == "" {
		return 0, ErrDocProjectURLRequired
	}

	doc, err := s.repo.GetDocumentByID(ctx, docID)
	if err != nil {
		return 0, err
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(origFilename), "."))
	if ext != "pdf" && ext != "doc" && ext != "docx" {
		return 0, ErrDocInvalidExt
	}

	if formType == "" {
		formType = "draft"
	}

	nextVer, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return 0, err
	}

	relPath := BuildDocRelPath(c, doc, doc.CurrentStage, formType, ext, nextVer)
	fullPath, err := SaveFileToUNC(c.ProjectURL, relPath, fileReader)
	if err != nil {
		return 0, err
	}

	ver := &models.DocumentVersion{
		DocumentID: doc.ID,
		VersionNum: nextVer,
		RelPath:    relPath,
		FileName:   filepath.Base(relPath),
		FileExt:    ext,
		FormType:   formType,
		Comment:    strings.TrimSpace(comment),
		UploadedBy: userID,
	}

	if err := s.repo.AddVersion(ctx, ver); err != nil {
		_ = os.Remove(fullPath) // откат файла при сбое БД
		return 0, err
	}

	return nextVer, nil
}

// GenerateFromTemplate формирует .docx из шаблона с автоподстановкой реквизитов договора и контрагента.
func (s *DocumentService) GenerateFromTemplate(
	ctx context.Context,
	contractID string,
	docID int64,
	templateName string,
	userID *int64,
) (int, error) {
	templateName = filepath.Base(strings.TrimSpace(templateName))
	if templateName == "" || templateName == "." {
		return 0, ErrDocTemplateRequired
	}

	c, err := s.contractRepo.GetByID(ctx, contractID)
	if err != nil {
		return 0, err
	}
	if c.ProjectURL == "" {
		return 0, ErrDocProjectURLRequired
	}

	doc, err := s.repo.GetDocumentByID(ctx, docID)
	if err != nil {
		return 0, err
	}

	cpName := ""
	if c.CounterpartyID != "" && s.cpRepo != nil {
		if cp, err := s.cpRepo.GetByID(ctx, c.CounterpartyID); err == nil && cp != nil {
			cpName = cp.Name
		}
	}

	contractDate := ""
	if c.ContractDate != nil {
		contractDate = c.ContractDate.Format("02.01.2006")
	}

	replacements := map[string]string{
		"{{CONTRACT_ID}}":       c.ID,
		"{{CONTRACT_NUMBER}}":   c.ContractNumber,
		"{{CONTRACT_DATE}}":     contractDate,
		"{{WORK_NAME}}":         c.WorkName,
		"{{SHORT_NAME}}":        c.ShortName,
		"{{COUNTERPARTY_NAME}}": cpName,
		"{{DOC_NUMBER}}":        doc.Number,
		"{{DOC_TITLE}}":         doc.Title,
		"{{TODAY}}":             time.Now().Format("02.01.2006"),
	}

	nextVer, err := s.repo.NextVersionNumber(ctx, doc.ID)
	if err != nil {
		return 0, err
	}

	relPath := BuildDocRelPath(c, doc, doc.CurrentStage, "draft", "docx", nextVer)
	destFullPath := filepath.Join(c.ProjectURL, relPath)
	templateFullPath := filepath.Join(s.templatesDir, templateName)

	if err := GenerateDocxFromTemplate(templateFullPath, destFullPath, replacements); err != nil {
		return 0, err
	}

	ver := &models.DocumentVersion{
		DocumentID: doc.ID,
		VersionNum: nextVer,
		RelPath:    relPath,
		FileName:   filepath.Base(relPath),
		FileExt:    "docx",
		FormType:   "draft",
		Comment:    "Из шаблона: " + templateName,
		UploadedBy: userID,
	}

	if err := s.repo.AddVersion(ctx, ver); err != nil {
		_ = os.Remove(destFullPath)
		return 0, err
	}

	return nextVer, nil
}

func (s *DocumentService) GetDocumentFull(ctx context.Context, docID int64) (*models.ContractDocument, error) {
	return s.repo.GetDocumentFull(ctx, docID)
}

// DeleteVersion удаляет промежуточную версию из БД (крайнюю удалять запрещено).
func (s *DocumentService) DeleteVersion(ctx context.Context, docID, verID int64) error {
	doc, err := s.repo.GetDocumentFull(ctx, docID)
	if err != nil {
		return err
	}
	if len(doc.Versions) > 0 && doc.Versions[0].ID == verID {
		return fmt.Errorf("нельзя удалить актуальную (крайнюю) версию документа")
	}
	return s.repo.DeleteVersion(ctx, verID)
}

// AddHistory записывает событие согласования/отправки и перемещает файл в папку нового этапа.
func (s *DocumentService) AddHistory(ctx context.Context, contractID string, h *models.DocumentHistory) error {
	h.Stage = strings.TrimSpace(h.Stage)
	h.FormType = strings.TrimSpace(h.FormType)
	h.RecipientInfo = strings.TrimSpace(h.RecipientInfo)
	h.Note = strings.TrimSpace(h.Note)

	if h.Stage == "" {
		h.Stage = "draft"
	}
	if h.FormType == "" {
		h.FormType = "draft"
	}

	if err := s.repo.AddHistory(ctx, h); err != nil {
		return err
	}

	// Перемещаем привязанную (крайнюю) версию в подпапку нового этапа или в корень при подписании
	if h.VersionID != nil && *h.VersionID > 0 {
		c, errC := s.contractRepo.GetByID(ctx, contractID)
		doc, errD := s.repo.GetDocumentByID(ctx, h.DocumentID)
		ver, errV := s.repo.GetVersionByID(ctx, *h.VersionID)

		if errC == nil && errD == nil && errV == nil && c.ProjectURL != "" {
			newRelPath := BuildDocRelPath(c, doc, h.Stage, h.FormType, ver.FileExt, ver.VersionNum)
			if err := MoveFileOnUNC(c.ProjectURL, ver.RelPath, newRelPath); err == nil {
				_ = s.repo.UpdateVersionPath(ctx, ver.ID, newRelPath, filepath.Base(newRelPath))
			}
		}
	}

	return nil
}

func (s *DocumentService) DeleteDocument(ctx context.Context, docID int64) error {
	return s.repo.DeleteDocument(ctx, docID)
}

func (s *DocumentService) ListDocxTemplates() []string {
	if s.templatesDir == "" {
		return nil
	}
	entries, err := os.ReadDir(s.templatesDir)
	if err != nil {
		return nil
	}
	var res []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".docx") && !strings.HasPrefix(e.Name(), "~$") {
			res = append(res, e.Name())
		}
	}
	return res
}

func (s *DocumentService) GetDocumentByID(ctx context.Context, docID int64) (*models.ContractDocument, error) {
	return s.repo.GetDocumentByID(ctx, docID)
}

func (s *DocumentService) UpdateDocument(ctx context.Context, d *models.ContractDocument) error {
	d.Title = strings.TrimSpace(d.Title)
	d.Number = strings.TrimSpace(d.Number)
	d.HolderInfo = strings.TrimSpace(d.HolderInfo)

	if d.Title == "" {
		return ErrDocTitleRequired
	}
	if d.DocType == "" {
		d.DocType = "main"
	}

	return s.repo.UpdateDocument(ctx, d)
}

// StageFolderName переводит код этапа в понятное имя подпапки на сервере.
// Для этапа "signed" возвращает пустую строку (файл кладётся в корень папки документа).
func StageFolderName(stage string) string {
	switch stage {
	case "signed":
		return ""
	case "draft":
		return "Подготовка_проекта"
	case "internal_review":
		return "Внутреннее_согласование"
	case "client_review":
		return "На_рассмотрении_у_контрагента"
	case "rework":
		return "На_доработке"
	case "signing":
		return "На_подписании"
	default:
		return "В_работе"
	}
}
