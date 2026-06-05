@echo off
setlocal

set GO_DIR=%~dp0.go\bin
set PATH=%GO_DIR%;%PATH%
set CGO_ENABLED=1

:: Find gcc in common locations
if exist "C:\mingw64\bin\gcc.exe" (
    set CC=C:\mingw64\bin\gcc.exe
) else if exist "C:\ProgramData\chocolatey\lib\mingw\tools\install\mingw64\bin\gcc.exe" (
    set CC=C:\ProgramData\chocolatey\lib\mingw\tools\install\mingw64\bin\gcc.exe
) else (
    where gcc >nul 2>&1
    if errorlevel 1 (
        echo [ERRO] MinGW-w64 GCC nao encontrado.
        echo Instala em C:\mingw64\bin\ ou adiciona ao PATH.
        echo Descarrega de: https://github.com/brechtsanders/winlibs_mingw/releases
        pause
        exit /b 1
    )
)

echo === Discord Quest Emulator - Build ===
echo.

go version

echo.
echo A compilar (primeira vez pode demorar 5-10 min)...
echo.

go build -o discord-quest-emulator.exe .

if %errorlevel% equ 0 (
    echo.
    echo ✓ Build concluido: discord-quest-emulator.exe
    dir /B discord-quest-emulator.exe
) else (
    echo.
    echo ✗ Erro no build
    pause
    exit /b %errorlevel%
)

endlocal
