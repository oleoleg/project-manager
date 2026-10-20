@echo off
chcp 65001 > nul
setlocal

cd /d "%~dp0"

echo === Быстрая сборка Агента ===

echo [1/2] Убиваем старые процессы...
taskkill /F /IM pm-agent.exe >nul 2>&1

echo [2/2] Сборка pm-agent.exe...
go build -ldflags "-H=windowsgui" -o "C:\Users\Shiryaev\AppData\Local\Project Manager Agent\pm-agent.exe" .\cmd\agent

if errorlevel 1 (
    echo [ОШИБКА] Сборка не удалась.
    pause
    exit /b 1
)

echo.
echo === Готово! Агент обновлен ===
echo.
pause