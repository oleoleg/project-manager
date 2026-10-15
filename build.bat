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
    echo.
    echo Правый клик на build.bat -^> "Запуск от имени администратора".
    echo.
    pause
    exit /b 1
)

REM --- Останавливаем службу и трей, чтобы освободить файлы ---
echo [1/7] Остановка службы и трея...
taskkill /F /IM project-manager-tray.exe >nul 2>&1
sc stop ProjectManager >nul 2>&1
timeout /t 3 /nobreak >nul

REM --- Чистим dist ---
echo [2/7] Очистка dist...
if exist dist rmdir /S /Q dist
mkdir dist

REM --- Включаем CGO ---
set CGO_ENABLED=1

REM --- Собираем службу ---
echo [3/7] Сборка project-manager.exe (служба)...
go build -ldflags "-H=windowsgui" -o dist\project-manager.exe .\cmd\app
if errorlevel 1 (
    echo [ОШИБКА] Сборка службы не удалась.
    pause
    exit /b 1
)

REM --- Собираем трей ---
echo [4/7] Сборка project-manager-tray.exe (трей)...
go build -ldflags "-H=windowsgui" -o dist\project-manager-tray.exe .\cmd\tray
if errorlevel 1 (
    echo [ОШИБКА] Сборка трея не удалась.
    pause
    exit /b 1
)

REM --- Собираем debug-версию ---
echo [5/7] Сборка project-manager-debug.exe...
go build -o dist\project-manager-debug.exe .\cmd\app
if errorlevel 1 (
    echo [ОШИБКА] Сборка debug-версии не удалась.
    pause
    exit /b 1
)

REM --- Готовим .env для dist ---
echo [6/7] Подготовка .env для dist...
if exist ".env.production" (
    copy /Y .env.production dist\.env >nul
    echo       .env.production -^> dist\.env
) else if exist ".env.production.example" (
    copy /Y .env.production.example dist\.env >nul
    echo       .env.production.example -^> dist\.env - не забудьте отредактировать!
) else (
    echo [ОШИБКА] Ни .env.production, ни .env.production.example не найдены.
    pause
    exit /b 1
)

REM --- Копируем web ---
echo [7/7] Копирование web...
if exist web (
    xcopy /E /I /Y /Q web dist\web >nul
    echo       web скопирован
) else (
    echo [ОШИБКА] Папка web не найдена.
    pause
    exit /b 1
)

echo.
echo === Сборка завершена ===
echo Бинарники: dist\project-manager.exe, dist\project-manager-tray.exe
echo.

pause