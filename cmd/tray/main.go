package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/energye/systray"
)

//go:embed assets/icon.ico
var iconBytes []byte

var (
	mStart   *systray.MenuItem
	mStop    *systray.MenuItem
	mRestart *systray.MenuItem

	cfg     Config
	stopCh  = make(chan struct{})
	cfgPath string
)

type Config struct {
	MonitorEnabled     bool `json:"monitor_enabled"`
	MonitorIntervalSec int  `json:"monitor_interval_sec"`
	Autostart          bool `json:"autostart"`
}

func main() {
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "pm://") {
		handleProtocolURL(os.Args[1])
		return
	}

	loadConfig()

	// Применяем автостарт по текущему значению конфига
	applyAutostart(cfg.Autostart)

	systray.Run(onReady, onExit)
}

// ---------------- Конфиг ----------------

func configPath() string {
	// %APPDATA%\ProjectManager\tray.config.json — доступно на запись
	// без прав администратора. Рядом с exe писать нельзя: exe лежит
	// в C:\Program Files, куда обычный пользователь не имеет доступа.
	appData := os.Getenv("APPDATA")
	if appData == "" {
		exe, _ := os.Executable()
		return filepath.Join(filepath.Dir(exe), "tray.config.json")
	}
	dir := filepath.Join(appData, "ProjectManager")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "tray.config.json")
}

func loadConfig() {
	cfgPath = configPath()

	cfg = Config{
		MonitorEnabled:     true,
		MonitorIntervalSec: 5,
		Autostart:          false,
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		// Файла нет — создадим с дефолтами
		saveConfig()
		return
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		// Битый JSON — оставляем дефолты
		return
	}

	// Санитарные ограничения
	if cfg.MonitorIntervalSec < 2 {
		cfg.MonitorIntervalSec = 2
	}
}

func saveConfig() {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(cfgPath, data, 0644)
}

// ---------------- Автостарт ----------------

// applyAutostart включает/выключает запись трея в автозагрузку пользователя.
func applyAutostart(enable bool) {
	exe, _ := os.Executable()
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	if enable {
		cmd := exec.Command("reg", "add", key, "/v", "ProjectManagerTray", "/t", "REG_SZ", "/d", exe, "/f")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	} else {
		cmd := exec.Command("reg", "delete", key, "/v", "ProjectManagerTray", "/f")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	}
}

// ---------------- Служба ----------------

func isServiceRunning() bool {
	cmd := exec.Command("sc", "query", "ProjectManager")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	s := string(out)
	return strings.Contains(s, "RUNNING") || strings.Contains(s, "РАБОТАЕТ")
}

// ---------------- Трей ----------------

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("Project Manager")
	systray.SetTooltip("Project Manager")

	mOpen := systray.AddMenuItem("Открыть в браузере", "")
	mOpen.Click(func() { openBrowser("http://localhost:8080") })

	systray.AddSeparator()

	mStart = systray.AddMenuItem("▶ Запустить службу", "")
	mStart.Click(func() { runSC("start") })

	mStop = systray.AddMenuItem("■ Остановить службу", "")
	mStop.Click(func() { runSC("stop") })

	mRestart = systray.AddMenuItem("↻ Перезапустить", "")
	mRestart.Click(func() {
		runSC("stop")
		time.Sleep(2 * time.Second)
		runSC("start")
	})

	systray.AddSeparator()

	mLogs := systray.AddMenuItem("Открыть лог", "")
	mLogs.Click(func() { openLogFile() })

	mLogsFolder := systray.AddMenuItem("Папка логов", "")
	mLogsFolder.Click(func() { openLogFolder() })

	mSettings := systray.AddMenuItem("⚙ Настройки", "Открыть файл tray.config.json")
	mSettings.Click(func() {
		// Открываем JSON в Блокноте. Пользователь правит, сохраняет.
		// Затем — «Применить настройки».
		_ = exec.Command("notepad.exe", cfgPath).Start()
	})

	mApply := systray.AddMenuItem("↻ Применить настройки", "Перечитать tray.config.json")
	mApply.Click(func() {
		loadConfig()
		applyAutostart(cfg.Autostart)
		restartMonitor()
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход (только трей)", "")
	mQuit.Click(func() {
		systray.Quit()
		os.Exit(0)
	})

	// Отложенный старт монитора
	go func() {
		time.Sleep(1 * time.Second)
		refreshMenuState()
		startMonitor()
	}()
}

func onExit() {}

// ---------------- Монитор состояния ----------------

func startMonitor() {
	if !cfg.MonitorEnabled {
		return
	}
	ticker := time.NewTicker(time.Duration(cfg.MonitorIntervalSec) * time.Second)
	go func() {
		for {
			select {
			case <-stopCh:
				ticker.Stop()
				return
			case <-ticker.C:
				refreshMenuState()
			}
		}
	}()
}

func restartMonitor() {
	select {
	case stopCh <- struct{}{}:
	default:
	}
	startMonitor()
}

func refreshMenuState() {
	running := isServiceRunning()

	if running {
		mStart.SetTitle("▶ Запустить службу (уже запущена)")
		mStop.SetTitle("■ Остановить службу")
		mRestart.SetTitle("↻ Перезапустить")
		mStart.Disable()
		mStop.Enable()
		mRestart.Enable()
	} else {
		mStart.SetTitle("▶ Запустить службу")
		mStop.SetTitle("■ Остановить службу (уже остановлена)")
		mRestart.SetTitle("↻ Перезапустить (служба не запущена)")
		mStart.Enable()
		mStop.Disable()
		mRestart.Disable()
	}
}

func runSC(action string) {
	running := isServiceRunning()
	if action == "start" && running {
		showMessage("Служба уже запущена.")
		return
	}
	if action == "stop" && !running {
		showMessage("Служба уже остановлена.")
		return
	}

	cmd := exec.Command("sc", action, "ProjectManager")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		showMessage("Ошибка: " + decodeCP866(string(out)))
		return
	}

	time.Sleep(1 * time.Second)
	refreshMenuState()
}

// ---------------- Вспомогательные ----------------

func handleProtocolURL(raw string) {
	u, err := url.Parse(raw)
	if err != nil {
		return
	}
	switch u.Host {
	case "open":
		path := u.Query().Get("path")
		if path == "" {
			return
		}
		_ = exec.Command("explorer", path).Start()
	case "pick-file":
		pickFileAndSavePath()
	}
}

// pickFileAndSavePath открывает стандартный Windows-диалог выбора файла,
// получает полный путь и сохраняет его в %APPDATA%\ProjectManager\picked_path.txt.
// Форма акта читает этот файл и подставляет путь в поле «Файл акта».
func pickFileAndSavePath() {
	path, err := openFileDialog()
	if err != nil || path == "" {
		return
	}

	appData := os.Getenv("APPDATA")
	if appData == "" {
		return
	}
	dir := filepath.Join(appData, "ProjectManager")
	_ = os.MkdirAll(dir, 0755)

	out := filepath.Join(dir, "picked_path.txt")
	_ = os.WriteFile(out, []byte(path), 0644)
}

func showMessage(text string) {
	tmp := filepath.Join(os.TempDir(), "pm-error.txt")
	_ = os.WriteFile(tmp, []byte(text), 0644)
	_ = exec.Command("notepad.exe", tmp).Start()
}

func decodeCP866(s string) string {
	cp866 := "АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдежзийклмнопрстуфхцчшщъыьэюя"
	utf := "АБВГДЕЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдежзийклмнопрстуфхцчшщъыьэюя"
	out := []rune{}
	for _, r := range s {
		if r < 128 {
			out = append(out, r)
			continue
		}
		idx := strings.IndexRune(cp866, r)
		if idx >= 0 {
			out = append(out, []rune(utf)[idx])
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func openLogFile() {
	_ = exec.Command("notepad.exe", logFilePath()).Start()
}

func openLogFolder() {
	_ = exec.Command("explorer", filepath.Dir(logFilePath())).Start()
}

func logFilePath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "logs", "project-manager.log")
}

// Заглушка для отладки импорта
var _ = fmt.Sprintf
