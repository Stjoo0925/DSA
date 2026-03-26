@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
powershell.exe -ExecutionPolicy Bypass -File "%SCRIPT_DIR%scripts\build-installer.ps1"
if errorlevel 1 exit /b %errorlevel%

echo.
echo Installer build completed.
endlocal
