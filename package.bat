@echo off
chcp 65001 > nul
setlocal

cd /d "%~dp0"

echo.
echo === Сборка Project Manager + инсталлятор ===
echo.

REM --- Сборка dist через build.bat (нужны права администратора) ---
call build.bat
if errorlevel 1 (
    echo [ОШИБКА] Сборка dist не удалась.
    pause
    exit /b 1
)

REM --- Сборка инсталлятора через Inno Setup ---
echo.
echo === Сборка инсталлятора ===

set ISCC="D:\Program Files\Inno Setup 7\ISCC.exe"

if not exist %ISCC% (
    echo [ОШИБКА] Inno Setup не найден по пути:
    echo         %ISCC%
    echo.
    echo Поправь путь в package.bat или установи Inno Setup 7.
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