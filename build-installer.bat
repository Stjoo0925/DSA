@echo off
setlocal

powershell -ExecutionPolicy Bypass -File "%~dp0scripts\build-installer.ps1"
if errorlevel 1 exit /b %errorlevel%

echo.
echo 설치 파일 생성 완료
endlocal
