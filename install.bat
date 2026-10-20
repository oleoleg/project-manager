@echo off
chcp 65001 > nul
setlocal EnableDelayedExpansion

cd /d "%~dp0"

echo.
echo === Установка службы Project Manager ===
echo.

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ОШИБКА] Скрипт нужно запускать ОТ ИМЕНИ АДМИНИСТРАТОРА.
    pause
    exit /b 1
)

if not exist "dist\project-manager.exe" (
    echo [ОШИБКА] dist\project-manager.exe не найден.
    pause
    exit /b 1
)

REM --- Удаляем старую версию службы, если есть ---
echo [1/7] Удаление старой службы (если есть)...
sc query ProjectManager >nul 2>&1
if %errorlevel% equ 0 (
    sc stop ProjectManager >nul 2>&1
    "dist\project-manager.exe" -remove >nul 2>&1
    timeout /t 2 /nobreak >nul
)

REM --- Устанавливаем новую ---
echo [2/7] Установка службы...
pushd dist
project-manager.exe -install
if errorlevel 1 (
    echo [ОШИБКА] Не удалось установить службу.
    popd
    pause
    exit /b 1
)
popd

REM --- Права на службу ---
echo [3/7] Настройка прав на службу...
sc sdset ProjectManager "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)(A;;RPWPCR;;;AU)" >nul 2>&1

REM --- Проверяем статус ---
echo [4/7] Проверка статуса...
timeout /t 2 /nobreak >nul
sc query ProjectManager | findstr "STATE"

REM --- Регистрируем URL-протокол pm-agent:// ---
echo [5/7] Регистрация протокола pm-agent://...
reg add "HKCR\pm-agent" /ve /t REG_SZ /d "URL:Project Manager Agent Protocol" /f >nul
reg add "HKCR\pm-agent" /v "URL Protocol" /t REG_SZ /d "" /f >nul
reg add "HKCR\pm-agent\shell\open\command" /ve /t REG_SZ /d "\"%~dp0dist\pm-agent.exe\" \"%%1\"" /f >nul

REM --- Убиваем старые процессы (Трей и Агент) ---
echo [6/7] Перезапускаем трей и агент...
taskkill /F /IM project-manager-tray.exe >nul 2>&1
taskkill /F /IM pm-agent.exe >nul 2>&1
timeout /t 1 /nobreak >nul

REM --- Запускаем Агент и Трей ---
start "" "dist\pm-agent.exe"
start "" "dist\project-manager-tray.exe"

echo.
echo === Установка завершена ===
echo Открыть веб-интерфейс: http://localhost:8080
echo.
pause