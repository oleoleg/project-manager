//go:build windows

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
	// Подключаем обертку
)

var (
	comdlg32             = syscall.NewLazyDLL("comdlg32.dll")
	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
)

const (
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_PATHMUSTEXIST = 0x00000800
	OFN_NOCHANGEDIR   = 0x00000008
	OFN_EXPLORER      = 0x00080000
	OFN_LONGNAMES     = 0x00200000
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	Flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	FlagsEx           uint32
}

// openFileDialog открывает стандартный Windows-диалог «Открыть файл».
// filterType: "pdf", "excel" или "" (все файлы).
func openFileDialog(initialPath string, filterType string) (string, error) {
	buf := make([]uint16, 4096)

	// Формируем фильтр в зависимости от переданного типа
	var filterStr string
	switch filterType {
	case "pdf":
		filterStr = "PDF файлы\x00*.pdf\x00Все файлы\x00*.*\x00\x00"
	case "excel":
		filterStr = "Excel файлы\x00*.xlsx;*.xls\x00Все файлы\x00*.*\x00\x00"
	default:
		filterStr = "Все файлы\x00*.*\x00\x00"
	}

	filterRunes := []uint16{}
	for _, part := range strings.Split(filterStr, "\x00") {
		u, _ := syscall.UTF16FromString(part)
		filterRunes = append(filterRunes, u...)
	}
	// Добавляем финальный двойной ноль
	filterRunes = append(filterRunes, 0)

	var initialDirPtr *uint16

	if initialPath != "" {
		dir := filepath.Dir(initialPath)
		if dir != "." && dir != "" {
			initialDirPtr, _ = syscall.UTF16PtrFromString(dir)
		}

		fileName := filepath.Base(initialPath)
		if fileName != "." && fileName != "" {
			u, _ := syscall.UTF16FromString(fileName)
			copy(buf, u)
		}
	}

	ofn := openFileNameW{
		lStructSize:     uint32(unsafe.Sizeof(openFileNameW{})),
		lpstrFilter:     &filterRunes[0],
		lpstrFile:       &buf[0],
		nMaxFile:        uint32(len(buf)),
		lpstrInitialDir: initialDirPtr,
		Flags:           OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR | OFN_EXPLORER | OFN_LONGNAMES,
	}

	ret, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return "", errors.New("file dialog cancelled")
	}

	return syscall.UTF16ToString(buf), nil
}

// --- GUID для IFileOpenDialog ---
var (
	CLSID_FileOpenDialog = syscall.GUID{Data1: 0xDC1C5A9C, Data2: 0xE8F2, Data3: 0x4D4C, Data4: [8]byte{0x90, 0x76, 0x5D, 0x3C, 0x1D, 0xFD, 0x7F, 0x8A}}
	IID_IFileOpenDialog  = syscall.GUID{Data1: 0xD57BDB6D, Data2: 0x4B4B, Data3: 0x4CB5, Data4: [8]byte{0xB7, 0x7A, 0x6E, 0xE4, 0xCA, 0x28, 0xC4, 0x0C}}
	// ВАЖНО: GUID выше может отличаться в зависимости от версии Windows.
	// Используем строковый CLSID для точности.
)

// openDirDialog открывает проверенный временем классический Windows-диалог выбора папки.
// func openDirDialog() (string, error) {
// 	buf := make([]uint16, 4096)

// 	// Стандартные флаги Windows для классического диалога
// 	const (
// 		BIF_RETURNONLYFSDIRS = 0x00000001 // Выбирать только папки файловой системы
// 		BIF_NEWDIALOGSTYLE   = 0x00000040 // Добавляет кнопку создания папок и делает окно растягиваемым
// 		BIF_EDITBOX          = 0x00000010 // Добавляет текстовое поле ввода пути сверху
// 	)

// 	type browseInfoW struct {
// 		hwndOwner      uintptr
// 		pidlRoot       uintptr
// 		pszDisplayName *uint16
// 		lpszTitle      *uint16
// 		ulFlags        uint32
// 		lpfn           uintptr
// 		lParam         uintptr
// 		iImage         int32
// 	}

// 	title, _ := syscall.UTF16PtrFromString("Выбор папки проекта")

// 	bi := browseInfoW{
// 		hwndOwner:      0,
// 		pidlRoot:       0,
// 		pszDisplayName: &buf[0],
// 		lpszTitle:      title,
// 		ulFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE | BIF_EDITBOX,
// 		lpfn:           0,
// 		lParam:         0,
// 		iImage:         0,
// 	}

// 	shell32 := syscall.NewLazyDLL("shell32.dll")
// 	procSHBrowseForFolderW := shell32.NewProc("SHBrowseForFolderW")
// 	procSHGetPathFromIDListW := shell32.NewProc("SHGetPathFromIDListW")

// 	// Открываем диалоговое окно
// 	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
// 	if pidl == 0 {
// 		return "", errors.New("folder dialog cancelled")
// 	}
// 	defer func() {
// 		ole32 := syscall.NewLazyDLL("ole32.dll")
// 		procCoTaskMemFree := ole32.NewProc("CoTaskMemFree")
// 		procCoTaskMemFree.Call(pidl)
// 	}()

// 	// Извлекаем текстовый путь из структуры Windows IDList
// 	ret, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
// 	if ret == 0 {
// 		return "", errors.New("could not get path from dialog")
// 	}

// 	return syscall.UTF16ToString(buf), nil
// }

// openDirDialog открывает современный диалог выбора папки через встроенный PowerShell.
func openDirDialog(initialDir string) (string, error) {
	// Скрипт PowerShell
	psScript := `
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.OpenFileDialog
$dialog.ValidateNames = $false
$dialog.CheckFileExists = $false
$dialog.CheckPathExists = $true
$dialog.FileName = "Выберите папку"
$dialog.Title = "Выберите папку проекта"
$dialog.Filter = "Папки|*.*"
`

	if initialDir != "" {
		// Экранируем одинарные кавычки для PowerShell
		safePath := strings.ReplaceAll(initialDir, "'", "''")
		psScript += fmt.Sprintf("$dialog.InitialDirectory = '%s'\n", safePath)
	}

	psScript += `
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    # Возвращаем папку, в которой находится выбранный "файл"
    Write-Output (Split-Path $dialog.FileName -Parent)
}
`

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	path := strings.TrimSpace(string(out))
	if path == "" || path == "Выберите папку" {
		return "", errors.New("folder dialog cancelled")
	}

	return path, nil
}
