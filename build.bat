@echo off
chcp 65001 > nul
setlocal

echo.
echo === Сборка Project Manager ===
echo.

REM --- Проверяем, что мы в корне проекта ---
if not exist "go.mod" (
    echo [ОШИБКА] go.mod не найден. Запустите скрипт из корня проекта.
    pause
    exit /b 1
)

REM --- Включаем CGO (нужен для systray) ---
set CGO_ENABLED=1

REM --- Чистим dist ---
echo [1/6] Очистка dist...
if exist dist rmdir /S /Q dist
mkdir dist

REM --- Собираем бинарник ---
echo [2/6] Сборка project-manager.exe...
go build -ldflags "-H=windowsgui" -o dist\project-manager.exe .\cmd\app
if errorlevel 1 (
    echo [ОШИБКА] Сборка не удалась.
    pause
    exit /b 1
)

REM --- Собираем трей ---
echo [3/6] Сборка project-manager-tray.exe (трей)...
go build -ldflags "-H=windowsgui" -o dist\project-manager-tray.exe .\cmd\tray
if errorlevel 1 exit /b 1

REM --- Собираем debug-версию (с консолью) ---
echo [4/6] Сборка project-manager-debug.exe...
go build -o dist\project-manager-debug.exe .\cmd\app
if errorlevel 1 (
    echo [ОШИБКА] Сборка debug-версии не удалась.
    pause
    exit /b 1
)

REM --- Копируем .env ---
echo [5/6] Копирование .env...
if exist .env (
    copy /Y .env dist\.env > nul
    echo       .env скопирован
) else (
    echo [ВНИМАНИЕ] .env не найден в корне проекта.
    echo            Создайте его из .env.example перед установкой.
)

REM --- Копируем web ---
echo [6/6] Копирование web...
if exist web (
    xcopy /E /I /Y /Q web dist\web > nul
    echo       web скопирован
) else (
    echo [ОШИБКА] Папка web не найдена.
    pause
    exit /b 1
)

echo.
echo === Сборка завершена ===
echo Бинарник: dist\project-manager.exe
echo.
dir dist
echo.

pause