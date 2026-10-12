@echo off
chcp 65001 > nul
setlocal

REM --- Всегда работаем из папки, где лежит сам скрипт ---
cd /d "%~dp0"


echo.
echo === Удаление службы Project Manager ===
echo.

REM --- Проверка прав администратора ---
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ОШИБКА] Скрипт нужно запускать ОТ ИМЕНИ АДМИНИСТРАТОРА.
    echo.
    pause
    exit /b 1
)

if not exist "dist\project-manager.exe" (
    echo [ОШИБКА] dist\project-manager.exe не найден.
    pause
    exit /b 1
)

echo [1/2] Остановка службы...
sc stop ProjectManager >nul 2>&1
timeout /t 2 /nobreak >nul

echo [2/2] Удаление службы...
dist\project-manager.exe -remove
if errorlevel 1 (
    echo [ОШИБКА] Не удалось удалить службу.
    pause
    exit /b 1
)

echo.
echo === Служба удалена ===
echo.
echo Иконка в трее может остаться - закройте её через правый клик - Выход.
echo.

pause