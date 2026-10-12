@echo off
chcp 65001 > nul
setlocal EnableDelayedExpansion

REM --- Всегда работаем из папки, где лежит сам скрипт ---
cd /d "%~dp0"

echo.
echo === Установка службы Project Manager ===
echo.

REM --- Проверка прав администратора ---
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ОШИБКА] Скрипт нужно запускать ОТ ИМЕНИ АДМИНИСТРАТОРА.
    echo.
    echo Правый клик на install.bat -^> "Запуск от имени администратора".
    echo.
    pause
    exit /b 1
)

REM --- Проверяем наличие dist ---
if not exist "dist\project-manager.exe" (
    echo [ОШИБКА] dist\project-manager.exe не найден.
    echo Рабочая директория: %CD%
    echo Сначала запустите build.bat.
    pause
    exit /b 1
)

if not exist "dist\.env" (
    echo [ОШИБКА] dist\.env не найден.
    echo Рабочая директория: %CD%
    echo Скопируйте .env в dist или запустите build.bat заново.
    pause
    exit /b 1
)

if not exist "dist\web\templates" (
    echo [ОШИБКА] dist\web\templates не найден.
    echo Рабочая директория: %CD%
    echo Запустите build.bat заново.
    pause
    exit /b 1
)

REM --- Удаляем старую версию службы, если есть ---
echo [1/4] Удаление старой службы (если есть)...
sc query ProjectManager >nul 2>&1
if %errorlevel% equ 0 (
    echo       Останавливаем...
    sc stop ProjectManager >nul 2>&1
    echo       Удаляем...
    "dist\project-manager.exe" -remove >nul 2>&1
    timeout /t 2 /nobreak >nul
)

REM --- Устанавливаем новую ---
echo [2/4] Установка службы...
pushd dist
project-manager.exe -install
if errorlevel 1 (
    echo [ОШИБКА] Не удалось установить службу.
    popd
    pause
    exit /b 1
)
popd

REM --- Проверяем статус ---
echo [3/4] Проверка статуса...
timeout /t 2 /nobreak >nul
sc query ProjectManager | findstr "STATE"

REM --- Убиваем старые треи ---
echo [4/4] Перезапускаем новый трей
taskkill /F /IM project-manager-tray.exe >nul 2>&1
timeout /t 1 /nobreak >nul

REM --- Запускаем новый трей (наследует права текущей сессии) ---
start "" "dist\project-manager-tray.exe"

echo.
echo === Установка завершена ===
echo Открыть веб-интерфейс: http://localhost:8080
echo.
pause