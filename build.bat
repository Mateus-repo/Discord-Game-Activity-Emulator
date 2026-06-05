@echo off
setlocal

set GO_DIR=%~dp0.go\bin
set PATH=%GO_DIR%;%PATH%

echo === Discord Quest Emulator - Build ===
echo.

go version

echo.
echo A compilar...
go build -o discord-quest-emulator.exe .

if %errorlevel% equ 0 (
    echo.
    echo ✓ Build concluido: discord-quest-emulator.exe
) else (
    echo.
    echo ✗ Erro no build
    exit /b %errorlevel%
)

endlocal
