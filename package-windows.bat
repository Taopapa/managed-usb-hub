@echo off
setlocal

cd /d "%~dp0"

echo.
echo ========================================
echo   C2G USB Hub Manager Windows Packager
echo ========================================
echo.

where go >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Go is not available in PATH.
  goto :fail
)

where wails >nul 2>nul
if errorlevel 1 (
  if exist "E:\go\bin\wails.exe" (
    set "PATH=E:\go\bin;%PATH%"
  ) else (
    echo [ERROR] Wails CLI is not available in PATH.
    echo Please make sure you have installed wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest
    goto :fail
  )
)

if not exist "build\bin" (
  mkdir "build\bin"
)

set "APP_VERSION=unknown"
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "(Get-Content -Raw 'wails.json' | ConvertFrom-Json).info.productVersion"`) do (
  set "APP_VERSION=%%i"
)

echo [INFO] Version: %APP_VERSION%
echo [INFO] Building CLI...
go build -o "build\bin\muhcli.exe" .\cmd\muhcli
if errorlevel 1 (
  echo [ERROR] Failed to build muhcli.exe
  goto :fail
)

echo [INFO] Building GUI and installer...
call wails build -platform windows/amd64 -nsis
if errorlevel 1 (
  echo [ERROR] Failed to build GUI installer.
  goto :fail
)

echo.
echo [SUCCESS] Packaging completed.
echo [OUTPUT] Installer: build\bin\C2G USB Hub Manager-amd64-installer.exe
echo [OUTPUT] GUI App  : build\bin\C2G USB Hub Manager.exe
echo [OUTPUT] CLI Tool : build\bin\muhcli.exe
goto :end

:fail
echo.
echo [FAILED] Packaging did not complete successfully.
pause
exit /b 1

:end
echo.
pause
