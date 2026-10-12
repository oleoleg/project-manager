@echo off
chcp 65001 > nul
setlocal

echo Ищем процессы на порту 8081...
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8081" ^| findstr "LISTENING"') do (
    echo Убиваем PID %%a
    taskkill /F /PID %%a >nul 2>&1
)

echo Готово.
timeout /t 2 /nobreak >nul