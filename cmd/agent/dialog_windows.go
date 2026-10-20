//go:build windows

package main

import (
	"errors"
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
func openFileDialog() (string, error) {
	buf := make([]uint16, 4096)

	filter := "Все файлы\x00*.*\x00PDF\x00*.pdf\x00\x00"
	filterUTF16, _ := syscall.UTF16FromString(filter)
	filterRunes := []uint16{}
	for _, part := range []string{"Все файлы", "*.*", "PDF", "*.pdf", ""} {
		u, _ := syscall.UTF16FromString(part)
		filterRunes = append(filterRunes, u...)
	}
	copy(filterUTF16, filterRunes)

	ofn := openFileNameW{
		lStructSize: uint32(unsafe.Sizeof(openFileNameW{})),
		lpstrFilter: &filterRunes[0],
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		Flags:       OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR | OFN_EXPLORER | OFN_LONGNAMES,
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

// openDirDialog открывает НАСТОЯЩИЙ современный диалог выбора папки Windows.
// func openDirDialog() (string, error) {
// 	// Создаем конфигурацию для диалога выбора папок
// 	config := cfd.DialogConfig{
// 		Title: "Выбор папки проекта",
// 		Role:  "ProjectFolderPicker", // Позволяет Windows запоминать последнюю выбранную папку
// 	}

// 	// Инициализируем современный нативный диалог
// 	dialog, err := cfd.NewSelectFolderDialog(config)
// 	if err != nil {
// 		return "", err
// 	}
// 	// Гарантируем освобождение системных ресурсов COM при выходе из функции
// 	defer dialog.Release()

// 	// Выводим полноценное окно Проводника на экран
// 	path, err := dialog.ShowAndGetResult()
// 	if err != nil {
// 		// Если пользователь просто закрыл окно крестиком или нажав "Отмена"
// 		if err.Error() == "The operation was canceled by the user." {
// 			return "", errors.New("folder dialog cancelled")
// 		}
// 		return "", err
// 	}

// 	return path, nil
// }

// openDirDialog открывает проверенный временем классический Windows-диалог выбора папки.
func openDirDialog() (string, error) {
	buf := make([]uint16, 4096)

	// Стандартные флаги Windows для классического диалога
	const (
		BIF_RETURNONLYFSDIRS = 0x00000001 // Выбирать только папки файловой системы
		BIF_NEWDIALOGSTYLE   = 0x00000040 // Добавляет кнопку создания папок и делает окно растягиваемым
		BIF_EDITBOX          = 0x00000010 // Добавляет текстовое поле ввода пути сверху
	)

	type browseInfoW struct {
		hwndOwner      uintptr
		pidlRoot       uintptr
		pszDisplayName *uint16
		lpszTitle      *uint16
		ulFlags        uint32
		lpfn           uintptr
		lParam         uintptr
		iImage         int32
	}

	title, _ := syscall.UTF16PtrFromString("Выбор папки проекта")

	bi := browseInfoW{
		hwndOwner:      0,
		pidlRoot:       0,
		pszDisplayName: &buf[0],
		lpszTitle:      title,
		ulFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE | BIF_EDITBOX,
		lpfn:           0,
		lParam:         0,
		iImage:         0,
	}

	shell32 := syscall.NewLazyDLL("shell32.dll")
	procSHBrowseForFolderW := shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW := shell32.NewProc("SHGetPathFromIDListW")

	// Открываем диалоговое окно
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", errors.New("folder dialog cancelled")
	}
	defer func() {
		ole32 := syscall.NewLazyDLL("ole32.dll")
		procCoTaskMemFree := ole32.NewProc("CoTaskMemFree")
		procCoTaskMemFree.Call(pidl)
	}()

	// Извлекаем текстовый путь из структуры Windows IDList
	ret, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
	if ret == 0 {
		return "", errors.New("could not get path from dialog")
	}

	return syscall.UTF16ToString(buf), nil
}
