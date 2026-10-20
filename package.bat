@echo off
chcp 65001 > nul
setlocal

cd /d "%~dp0"

echo.
echo === Сборка Project Manager + инсталляторы ===
echo.

REM --- Сборка dist через build.bat ---
call build.bat
if errorlevel 1 (
    echo [ОШИБКА] Сборка dist не удалась.
    pause
    exit /b 1
)

echo.
echo === Сборка инсталляторов ===

set ISCC="D:\Program Files\Inno Setup 7\ISCC.exe"

if not exist %ISCC% (
    echo [ОШИБКА] Inno Setup не найден по пути: %ISCC%
    pause
    exit /b 1
)

echo [1/2] Сборка серверного инсталлятора...
%ISCC% installer\setup-server.iss
if errorlevel 1 (
    echo [ОШИБКА] Не удалось собрать setup-server.iss
    pause
    exit /b 1
)

echo [2/2] Сборка клиентского инсталлятора (Агент)...
%ISCC% installer\setup-agent.iss
if errorlevel 1 (
    echo [ОШИБКА] Не удалось собрать setup-agent.iss
    pause
    exit /b 1
)

echo.
echo === Готово ===
echo Серверный инсталлятор: dist\ProjectManager-Server-Setup.exe
echo Клиентский инсталлятор: dist\ProjectManager-Agent-Setup.exe
echo.

pause