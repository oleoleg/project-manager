@echo off
chcp 65001 > nul
setlocal EnableDelayedExpansion

REM --- Всегда работаем из папки, где лежит сам скрипт ---
cd /d "%~dp0"

echo.
echo === Сборка Project Manager ===
echo.

REM --- Проверяем, что мы в корне проекта ---
if not exist "go.mod" (
    echo [ОШИБКА] go.mod не найден. Запустите скрипт из корня проекта.
    pause
    exit /b 1
)

REM --- Проверка прав администратора (нужна для остановки службы) ---
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ОШИБКА] Скрипт нужно запускать ОТ ИМЕНИ АДМИНИСТРАТОРА.
    pause
    exit /b 1
)

REM --- Останавливаем службу, трей и агент, чтобы освободить файлы ---
echo [1/9] Остановка службы, трея и агента...
taskkill /F /IM project-manager-tray.exe >nul 2>&1
taskkill /F /IM pm-agent.exe >nul 2>&1
taskkill /F /IM pm-agent-debug.exe >nul 2>&1
taskkill /F /IM project-manager.exe >nul 2>&1
taskkill /F /IM project-manager-debug.exe >nul 2>&1
sc stop ProjectManager >nul 2>&1
timeout /t 3 /nobreak >nul

REM --- Чистим кэш Go ---
echo [2/9] Очистка кэша Go...
go clean -cache

REM --- Чистим dist ---
echo [3/9] Очистка dist...
if exist dist rmdir /S /Q dist
mkdir dist

REM --- Включаем CGO ---
set CGO_ENABLED=1

REM --- Собираем службу ---
echo [4/9] Сборка project-manager.exe (служба)...
go build -a -ldflags "-H=windowsgui" -o dist\project-manager.exe .\cmd\app
if errorlevel 1 (
    echo [ОШИБКА] Сборка службы не удалась.
    pause
    exit /b 1
)

REM --- Собираем трей (Админ) ---
echo [5/9] Сборка project-manager-tray.exe (трей)...
go build -a -ldflags "-H=windowsgui" -o dist\project-manager-tray.exe .\cmd\tray
if errorlevel 1 (
    echo [ОШИБКА] Сборка трея не удалась.
    pause
    exit /b 1
)

REM --- Собираем Агент (Клиент) ---
echo [6/9] Сборка pm-agent.exe (агент)...
go build -a -ldflags "-H=windowsgui" -o dist\pm-agent.exe .\cmd\agent
if errorlevel 1 (
    echo [ОШИБКА] Сборка агента не удалась.
    pause
    exit /b 1
)

REM --- Собираем debug-версию ---
echo [7/9] Сборка project-manager-debug.exe...
go build -a -o dist\project-manager-debug.exe .\cmd\app
if errorlevel 1 (
    echo [ОШИБКА] Сборка debug-версии не удалась.
    pause
    exit /b 1
)

REM --- Готовим .env для dist ---
echo [8/9] Подготовка .env для dist...
if exist ".env.production" (
    copy /Y .env.production dist\.env >nul
) else if exist ".env.production.example" (
    copy /Y .env.production.example dist\.env >nul
) else (
    echo [ОШИБКА] Ни .env.production, ни .env.production.example не найдены.
    pause
    exit /b 1
)

REM --- Копируем web ---
echo [9/9] Копирование web...
if exist web (
    xcopy /E /I /Y /Q web dist\web >nul
) else (
    echo [ОШИБКА] Папка web не найдена.
    pause
    exit /b 1
)

echo.
echo === Сборка завершена ===
echo Бинарники лежат в папке dist\
echo.

pause