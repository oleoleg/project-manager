package main

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/energye/systray"
)

//go:embed assets/icon.ico
var iconBytes []byte

var (
	cfg     Config
	cfgPath string
)

// Config для агента
type Config struct {
	Autostart bool   `json:"autostart"`
	ServerURL string `json:"server_url"`
}

func main() {
	// 1. Обработка вызова из браузера (pm-agent://...)
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "pm-agent://") {
		handleProtocolURL(os.Args[1])
		return
	}

	// 2. Запуск в трее
	loadConfig()
	applyAutostart(cfg.Autostart)
	systray.Run(onReady, onExit)
}

// ---------------- Конфиг ----------------

func configPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		exe, _ := os.Executable()
		return filepath.Join(filepath.Dir(exe), "agent.config.json")
	}
	dir := filepath.Join(appData, "ProjectManager")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "agent.config.json")
}

func loadConfig() {
	cfgPath = configPath()

	cfg = Config{
		Autostart: true,
		ServerURL: "http://localhost:8080",
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		saveConfig()
		return
	}
	_ = json.Unmarshal(data, &cfg)
}

func saveConfig() {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(cfgPath, data, 0644)
}

// ---------------- Автостарт ----------------

func applyAutostart(enable bool) {
	exe, _ := os.Executable()
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	cmd := exec.Command("reg")
	if enable {
		cmd = exec.Command("reg", "add", key, "/v", "ProjectManagerAgent", "/t", "REG_SZ", "/d", exe, "/f")
	} else {
		cmd = exec.Command("reg", "delete", key, "/v", "ProjectManagerAgent", "/f")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}

// ---------------- Трей ----------------

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("Project Manager Agent")
	systray.SetTooltip("Агент доступа к файлам Project Manager")

	mOpen := systray.AddMenuItem("Открыть в браузере", cfg.ServerURL)
	mOpen.Click(func() { openBrowser(cfg.ServerURL) })

	systray.AddSeparator()

	mSettings := systray.AddMenuItem("⚙ Настройки", "Открыть файл конфигурации")
	mSettings.Click(func() {
		_ = exec.Command("notepad.exe", cfgPath).Start()
	})

	mApply := systray.AddMenuItem("↻ Применить настройки", "Перечитать конфигурацию")
	mApply.Click(func() {
		loadConfig()
		applyAutostart(cfg.Autostart)
		mOpen.SetTitle("Открыть в браузере")
		mOpen.SetTooltip(cfg.ServerURL)
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход", "Закрыть агент")
	mQuit.Click(func() {
		systray.Quit()
		os.Exit(0)
	})
}

func onExit() {}

// ---------------- Обработка протокола ----------------

func handleProtocolURL(raw string) {
	// 1. Убираем протокол
	cleanURL := strings.TrimPrefix(raw, "pm-agent://")

	// 2. Убираем слэши в начале и в конце
	cleanURL = strings.Trim(cleanURL, "/")

	// 3. Обрабатываем команду open (поддерживаем и open?, и open/?)
	if strings.HasPrefix(cleanURL, "open?") || strings.HasPrefix(cleanURL, "open/?") {
		idx := strings.Index(cleanURL, "path=")
		if idx != -1 {
			path := cleanURL[idx+5:]
			if unescaped, err := url.QueryUnescape(path); err == nil {
				path = unescaped
			}
			if path != "" {
				_ = exec.Command("explorer", path).Start()
			}
		}
	} else if strings.HasPrefix(cleanURL, "pick-file") {
		pickFileAndSavePath()
	} else if strings.HasPrefix(cleanURL, "pick-dir") {
		pickDirAndSavePath()
	}
}

// ---------------- Диалоги и утилиты ----------------

func pickFileAndSavePath() {
	path, err := openFileDialog()
	if err != nil || path == "" {
		return
	}
	savePickedPath(path)
}

func pickDirAndSavePath() {
	path, err := openDirDialog()
	if err != nil || path == "" {
		return
	}
	savePickedPath(path)
}

// func pickDirAndSavePath() {
// 	// Используем современный диалог выбора ФАЙЛА
// 	path, err := openFileDialog()
// 	if err != nil || path == "" {
// 		return
// 	}

// 	// Отрезаем имя файла, оставляя только папку
// 	dir := filepath.Dir(path)

//		savePickedPath(dir)
//	}

func savePickedPath(path string) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return
	}
	dir := filepath.Join(appData, "ProjectManager")
	_ = os.MkdirAll(dir, 0755)
	out := filepath.Join(dir, "picked_path.txt")
	_ = os.WriteFile(out, []byte(path), 0644)
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
