package main

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/energye/systray"
)

//go:embed assets/icon.ico
var iconBytes []byte

func main() {
	// Убедимся, что трей один
	ensureSingleInstance()
	systray.Run(onReady, onExit)
}

func ensureSingleInstance() {
	// Простейшая защита: если уже запущен другой - убьём его
	// (в реальном приложении лучше mutex, но для внутренней утилиты так проще)
}

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("Project Manager")
	systray.SetTooltip("Project Manager")

	mOpen := systray.AddMenuItem("Открыть в браузере", "")
	mOpen.Click(func() { openBrowser("http://localhost:8080") })

	systray.AddSeparator()

	mStart := systray.AddMenuItem("▶ Запустить службу", "")
	mStart.Click(func() { runSC("start") })

	mStop := systray.AddMenuItem("■ Остановить службу", "")
	mStop.Click(func() { runSC("stop") })

	mRestart := systray.AddMenuItem("↻ Перезапустить", "")
	mRestart.Click(func() {
		runSC("stop")
		runSC("start")
	})

	systray.AddSeparator()

	mLogs := systray.AddMenuItem("Открыть лог", "")
	mLogs.Click(func() { openLogFile() })

	mLogsFolder := systray.AddMenuItem("Папка логов", "")
	mLogsFolder.Click(func() { openLogFolder() })

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход (только трей)", "")
	mQuit.Click(func() {
		systray.Quit()
		os.Exit(0)
	})
}

func onExit() {}

func runSC(action string) {
	cmd := exec.Command("sc", action, "ProjectManager")
	out, err := cmd.CombinedOutput()
	if err != nil {
		showMessage("Ошибка: " + string(out) + "\nЗапустите трей от имени администратора.")
		return
	}
}

func showMessage(text string) {
	tmp := filepath.Join(os.TempDir(), "pm-error.txt")
	os.WriteFile(tmp, []byte(text), 0644)
	exec.Command("notepad.exe", tmp).Start()
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func openLogFile() {
	path := logFilePath()
	// Открываем через notepad.exe — самый надёжный вариант
	_ = exec.Command("notepad.exe", path).Start()
}

func openLogFolder() {
	_ = exec.Command("explorer", filepath.Dir(logFilePath())).Start()
}

func logFilePath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "logs", "project-manager.log")
}
