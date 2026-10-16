//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"
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
// Возвращает полный путь к выбранному файлу.
func openFileDialog() (string, error) {
	buf := make([]uint16, 4096)

	filter := "Все файлы\x00*.*\x00PDF\x00*.pdf\x00\x00"
	filterUTF16, _ := syscall.UTF16FromString(filter)
	// UTF16FromString добавляет \x00 в конце — но нам нужны двойные \x00,
	// поэтому соберём вручную
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
