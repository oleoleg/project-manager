@echo off
chcp 65001 > nul
setlocal

cd /d "%~dp0"

echo.
echo === Сборка Project Manager + инсталлятор ===
echo.

REM --- Сборка dist через build.bat (нужны права админа) ---
call build.bat
if errorlevel 1 (
    echo [ОШИБКА] Сборка dist не удалась.
    pause
    exit /b 1
)

REM --- Сборка инсталлятора через Inno Setup ---
echo.
echo === Сборка инсталлятора ===
set ISCC="C:\Program Files (x86)\Inno Setup 6\ISCC.exe"

if not exist %ISCC% (
    echo [ОШИБКА] Inno Setup не найден: %ISCC%
    echo Установи Inno Setup или поправь путь в package.bat.
    pause
    exit /b 1
)

%ISCC% installer\setup.iss
if errorlevel 1 (
    echo [ОШИБКА] Inno Setup не смог собрать инсталлятор.
    pause
    exit /b 1
)

echo.
echo === Готово ===
echo Инсталлятор: dist\ProjectManager-Setup.exe
echo.
pause