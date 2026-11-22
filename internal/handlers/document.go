package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type DocumentHandler struct {
	docSvc      *services.DocumentService
	contractSvc *services.ContractService
	cpSvc       *services.CounterpartyService
	tmpl        *template.Template
}

func NewDocumentHandler(
	docSvc *services.DocumentService,
	contractSvc *services.ContractService,
	cpSvc *services.CounterpartyService,
	tmpl *template.Template,
) *DocumentHandler {
	return &DocumentHandler{
		docSvc:      docSvc,
		contractSvc: contractSvc,
		cpSvc:       cpSvc,
		tmpl:        tmpl,
	}
}

func (h *DocumentHandler) Index(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	c, err := h.contractSvc.GetByID(r.Context(), contractID)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("documents index contract: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	docs, err := h.docSvc.ListByContract(r.Context(), c.ID)
	if err != nil {
		log.Printf("documents list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	users, _ := h.contractSvc.ListActiveUsers(r.Context())

	data := ContractBaseData(r, "Документы договора "+c.ID, c, h.cpSvc)
	data["CanEdit"] = CanEditContract(r, c)
	data["Documents"] = docs
	data["DocTypes"] = models.GetDocTypes()
	data["DocStages"] = models.GetDocStages()
	data["DocForms"] = models.GetDocForms()
	data["ActiveUsers"] = users
	data["DocxTemplates"] = h.docSvc.ListDocxTemplates()
	data["Flash"] = r.URL.Query().Get("flash")
	data["Error"] = r.URL.Query().Get("err")

	data["DocTypeName"] = func(code string) string { return findRefItemName(models.GetDocTypes(), code) }
	data["DocStageName"] = func(code string) string { return findRefItemName(models.GetDocStages(), code) }
	data["DocFormName"] = func(code string) string { return findRefItemName(models.GetDocForms(), code) }

	RenderTemplate(w, h.tmpl, "documents_list.html", data)
}

// Show — детальная страница одного документа (/contracts/{id}/documents/{docID})
func (h *DocumentHandler) Show(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, err := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	c, err := h.contractSvc.GetByID(r.Context(), contractID)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	doc, err := h.docSvc.GetDocumentFull(r.Context(), docID)
	if errors.Is(err, db.ErrDocumentNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	users, _ := h.contractSvc.ListActiveUsers(r.Context())

	var latestVer *models.DocumentVersion
	if len(doc.Versions) > 0 {
		latestVer = &doc.Versions[0]
	}

	data := ContractBaseData(r, "Документ "+doc.Title, c, h.cpSvc)
	data["CanEdit"] = CanEditContract(r, c)
	data["Document"] = doc
	data["LatestVersion"] = latestVer
	data["DocTypes"] = models.GetDocTypes()
	data["DocStages"] = models.GetDocStages()
	data["DocForms"] = models.GetDocForms()
	data["ActiveUsers"] = users
	data["DocxTemplates"] = h.docSvc.ListDocxTemplates()
	data["Flash"] = r.URL.Query().Get("flash")
	data["Error"] = r.URL.Query().Get("err")

	data["DocTypeName"] = func(code string) string { return findRefItemName(models.GetDocTypes(), code) }
	data["DocStageName"] = func(code string) string { return findRefItemName(models.GetDocStages(), code) }
	data["DocFormName"] = func(code string) string { return findRefItemName(models.GetDocForms(), code) }

	RenderTemplate(w, h.tmpl, "documents_show.html", data)
}

// CreateDocument — создание документа сразу с файлом/шаблоном и стартовой отметкой
func (h *DocumentHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	c, err := h.contractSvc.GetByID(r.Context(), contractID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	_ = r.ParseMultipartForm(50 << 20)

	stage := strings.TrimSpace(r.FormValue("stage"))
	if stage == "" {
		stage = "draft"
	}
	formType := strings.TrimSpace(r.FormValue("form_type"))
	if formType == "" {
		formType = "draft"
	}
	holder := strings.TrimSpace(r.FormValue("holder_info"))

	doc := &models.ContractDocument{
		ContractID:   contractID,
		DocType:      strings.TrimSpace(r.FormValue("doc_type")),
		Number:       strings.TrimSpace(r.FormValue("number")),
		Title:        strings.TrimSpace(r.FormValue("title")),
		CurrentStage: stage,
		CurrentForm:  formType,
		HolderInfo:   holder,
	}

	if err := h.docSvc.CreateDocument(r.Context(), doc); err != nil {
		h.renderDocForm(w, r, c, doc, true, err.Error())
		return
	}

	uid := currentUserID(r)
	var createdVerID *int64

	// 1. Либо загружен файл, либо выбран шаблон Word
	file, header, fileErr := r.FormFile("file")
	if fileErr == nil && header != nil && header.Filename != "" {
		defer file.Close()
		_, _ = h.docSvc.UploadVersion(r.Context(), contractID, doc.ID, header.Filename, file, formType, r.FormValue("comment"), uid)
	} else if tplName := strings.TrimSpace(r.FormValue("template_name")); tplName != "" {
		_, _ = h.docSvc.GenerateFromTemplate(r.Context(), contractID, doc.ID, tplName, uid)
	}

	// Получаем ID созданной v1 (если файл был создан), чтобы привязать к стартовой отметке
	if fullDoc, err := h.docSvc.GetDocumentFull(r.Context(), doc.ID); err == nil && len(fullDoc.Versions) > 0 {
		createdVerID = &fullDoc.Versions[0].ID
	}

	// 2. Если указана стартовая отметка о согласовании / отправке — записываем в историю
	if holder != "" || stage != "draft" || r.FormValue("note") != "" {
		_ = h.docSvc.AddHistory(r.Context(), contractID, &models.DocumentHistory{
			DocumentID:    doc.ID,
			VersionID:     createdVerID,
			Stage:         stage,
			FormType:      formType,
			RecipientInfo: holder,
			Note:          r.FormValue("note"),
			CreatedBy:     uid,
		})
	}

	http.Redirect(w, r, fmt.Sprintf("/contracts/%s/documents/%d?flash=%s", contractID, doc.ID, url.QueryEscape("Документ создан")), http.StatusSeeOther)
}

// DeleteVersion — удаление промежуточной версии из БД
func (h *DocumentHandler) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, _ := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	verID, _ := strconv.ParseInt(chi.URLParam(r, "verID"), 10, 64)

	if err := h.docSvc.DeleteVersion(r.Context(), docID, verID); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/contracts/%s/documents/%d?err=%s", contractID, docID, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/contracts/%s/documents/%d?flash=%s", contractID, docID, url.QueryEscape("Запись о версии удалена из базы")), http.StatusSeeOther)
}

func (h *DocumentHandler) OpenVersion(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	verID, err := strconv.ParseInt(chi.URLParam(r, "verID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	c, err := h.contractSvc.GetByID(r.Context(), contractID)
	if err != nil || c.ProjectURL == "" {
		http.Error(w, "Путь к корню проекта не задан", http.StatusNotFound)
		return
	}

	ver, err := h.docSvc.GetVersionByID(r.Context(), verID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(c.ProjectURL, ver.RelPath)
	target := "pm-agent://open?path=" + url.QueryEscape(fullPath)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="ru"><head><meta charset="utf-8"><title>Открытие файла…</title></head>
<body>
<script>
    window.location.href = %q;
    setTimeout(function(){ window.close(); }, 500);
</script>
<p>Открываем файл %s…</p>
</body></html>`, target, template.HTMLEscapeString(ver.FileName))
}

func (h *DocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, err := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.docSvc.DeleteDocument(r.Context(), docID); err != nil {
		log.Printf("delete document: %v", err)
		redirectDocErr(w, r, contractID, "Ошибка удаления")
		return
	}

	redirectDocFlash(w, r, contractID, "Документ удалён из реестра")
}

// ---------- helpers ----------

func currentUserID(r *http.Request) *int64 {
	if u := middleware.UserFromContext(r.Context()); u != nil {
		return &u.ID
	}
	return nil
}

func findRefItemName(list []models.RefItem, code string) string {
	for _, item := range list {
		if item.Code == code {
			return item.Name
		}
	}
	return code
}

func redirectDocFlash(w http.ResponseWriter, r *http.Request, contractID, msg string) {
	http.Redirect(w, r, fmt.Sprintf("/contracts/%s/documents?flash=%s", contractID, url.QueryEscape(msg)), http.StatusSeeOther)
}

func redirectDocErr(w http.ResponseWriter, r *http.Request, contractID, msg string) {
	http.Redirect(w, r, fmt.Sprintf("/contracts/%s/documents?err=%s", contractID, url.QueryEscape(msg)), http.StatusSeeOther)
}

// New — форма создания карточки документа (/contracts/{id}/documents/new)
func (h *DocumentHandler) New(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	c, err := h.contractSvc.GetByID(r.Context(), contractID)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("document new contract: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	doc := &models.ContractDocument{
		ContractID: c.ID,
		DocType:    "main",
	}
	h.renderDocForm(w, r, c, doc, true, "")
}

// Edit — форма редактирования карточки документа (/contracts/{id}/documents/{docID}/edit)
func (h *DocumentHandler) Edit(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, err := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	c, err := h.contractSvc.GetByID(r.Context(), contractID)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("document edit contract: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	doc, err := h.docSvc.GetDocumentByID(r.Context(), docID)
	if errors.Is(err, db.ErrDocumentNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("document edit get: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.renderDocForm(w, r, c, doc, false, "")
}

// Update — сохранение изменений карточки документа (POST /contracts/{id}/documents/{docID})
func (h *DocumentHandler) Update(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, err := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	c, err := h.contractSvc.GetByID(r.Context(), contractID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	_ = r.ParseForm()
	doc := &models.ContractDocument{
		ID:         docID,
		ContractID: contractID,
		DocType:    strings.TrimSpace(r.FormValue("doc_type")),
		Number:     strings.TrimSpace(r.FormValue("number")),
		Title:      strings.TrimSpace(r.FormValue("title")),
		HolderInfo: strings.TrimSpace(r.FormValue("holder_info")),
	}

	if err := h.docSvc.UpdateDocument(r.Context(), doc); err != nil {
		h.renderDocForm(w, r, c, doc, false, err.Error())
		return
	}

	redirectDocShowFlash(w, r, contractID, docID, "Изменения сохранены")
}

func (h *DocumentHandler) UploadVersion(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, err := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		redirectDocShowErr(w, r, contractID, docID, "Файл слишком большой (макс. 50 МБ)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		redirectDocShowErr(w, r, contractID, docID, "Выберите файл для загрузки")
		return
	}
	defer file.Close()

	verNum, err := h.docSvc.UploadVersion(
		r.Context(),
		contractID,
		docID,
		header.Filename,
		file,
		r.FormValue("form_type"),
		r.FormValue("comment"),
		currentUserID(r),
	)
	if err != nil {
		log.Printf("upload version: %v", err)
		redirectDocShowErr(w, r, contractID, docID, err.Error())
		return
	}

	redirectDocShowFlash(w, r, contractID, docID, fmt.Sprintf("Загружена версия v%d", verNum))
}

func (h *DocumentHandler) GenerateFromTemplate(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, err := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	_ = r.ParseForm()
	verNum, err := h.docSvc.GenerateFromTemplate(
		r.Context(),
		contractID,
		docID,
		r.FormValue("template_name"),
		currentUserID(r),
	)
	if err != nil {
		log.Printf("generate docx: %v", err)
		redirectDocShowErr(w, r, contractID, docID, err.Error())
		return
	}

	redirectDocShowFlash(w, r, contractID, docID, fmt.Sprintf("Документ сформирован (v%d)", verNum))
}

func (h *DocumentHandler) AddHistory(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	docID, err := strconv.ParseInt(chi.URLParam(r, "docID"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	_ = r.ParseForm()
	var versionID *int64
	if vStr := r.FormValue("version_id"); vStr != "" {
		if vID, err := strconv.ParseInt(vStr, 10, 64); err == nil && vID > 0 {
			versionID = &vID
		}
	}

	hist := &models.DocumentHistory{
		DocumentID:    docID,
		VersionID:     versionID,
		Stage:         r.FormValue("stage"),
		FormType:      r.FormValue("form_type"),
		RecipientInfo: r.FormValue("recipient_info"),
		Note:          r.FormValue("note"),
		CreatedBy:     currentUserID(r),
	}

	// Передаём contractID, чтобы сервис мог переместить файл на UNC-сервере в папку нового этапа
	if err := h.docSvc.AddHistory(r.Context(), contractID, hist); err != nil {
		log.Printf("add document history: %v", err)
		redirectDocShowErr(w, r, contractID, docID, "Не удалось сохранить отметку")
		return
	}

	redirectDocShowFlash(w, r, contractID, docID, "Статус обновлён")
}

func (h *DocumentHandler) renderDocForm(w http.ResponseWriter, r *http.Request, c *models.Contract, doc *models.ContractDocument, isNew bool, errMsg string) {
	title := "Редактирование документа"
	if isNew {
		title = "Новый документ по договору " + c.ID
	}

	users, _ := h.contractSvc.ListActiveUsers(r.Context())

	data := ContractBaseData(r, title, c, h.cpSvc)
	data["Document"] = doc
	data["DocTypes"] = models.GetDocTypes()
	data["DocStages"] = models.GetDocStages()
	data["DocForms"] = models.GetDocForms()
	data["ActiveUsers"] = users
	data["DocxTemplates"] = h.docSvc.ListDocxTemplates()
	data["IsNew"] = isNew
	data["Error"] = errMsg

	if errMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	RenderTemplate(w, h.tmpl, "documents_form.html", data)
}

func redirectDocShowFlash(w http.ResponseWriter, r *http.Request, contractID string, docID int64, msg string) {
	http.Redirect(w, r, fmt.Sprintf("/contracts/%s/documents/%d?flash=%s", contractID, docID, url.QueryEscape(msg)), http.StatusSeeOther)
}

func redirectDocShowErr(w http.ResponseWriter, r *http.Request, contractID string, docID int64, msg string) {
	http.Redirect(w, r, fmt.Sprintf("/contracts/%s/documents/%d?err=%s", contractID, docID, url.QueryEscape(msg)), http.StatusSeeOther)
}
