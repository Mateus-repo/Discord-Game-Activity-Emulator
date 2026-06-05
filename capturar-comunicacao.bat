@echo off
title Captura Discord Quest - Arknights Endfield
chcp 65001 >nul
cd /d "%~dp0"

echo === Captura de Comunicacao Discord Quest ===
echo.
echo Metodo 1: Discord DevTools (RECOMENDADO)
echo ==========================================
echo.
echo 1. Abre o Discord
echo 2. Carrega Ctrl+Shift+I para abrir DevTools
echo 3. Vai a tab "Network"
echo 4. Filtro: "quest" (para ver so pedidos de quests)
echo.
echo 5. Mantem o DevTools aberto e lanca o jogo
echo 6. Quando o Discord detetar o jogo, deves ver pedidos a aparecer
echo 7. Clica num pedido "heartbeat" ou parecido
echo 8. Vai a tab "Payload" para ver o corpo do pedido
echo 9. Direito do rato -^> Copy -^> Copy as cURL
echo 10. Cola o resultado aqui no chat
echo.
echo ==========================================
echo.
echo Metodo 2: netsh trace (alternativa)
echo ==========================================
echo.
echo 1. Fecha o Discord completamente
echo 2. Neste .bat, carrega numa tecla para iniciar captura
echo 3. Abre o Discord e depois o jogo
echo 4. Volta a este .bat e carrega noutra tecla para parar
echo 5. O ficheiro .etl pode ser analisado com Wireshark
echo.
echo ==========================================
echo.
echo A carregar... faz o Metodo 1 (DevTools) primeiro!
echo.
pause

echo.
echo A preparar captura de rede com netsh...
set TRACE_FILE=%USERPROFILE%\Desktop\discord_quest_capture_%DATE:~-4,4%%DATE:~-7,2%%DATE:~-10,2%_%TIME:~0,2%%TIME:~3,2%%TIME:~6,2%.etl
set TRACE_FILE=%TRACE_FILE: =0%

echo.
echo Ficheiro: %TRACE_FILE%
echo.
echo Passos:
echo 1. Carrega numa tecla para INICIAR captura
echo 2. Abre o Discord (se nao estiver aberto)
echo 3. Faz login se necessario
echo 4. Lanca o Arknights Endfield
echo 5. Espera 30-60s para o Discord detetar o jogo
echo 6. Volta aqui e carrega para PARAR captura
echo.
pause

echo.
echo A INICIAR captura...
netsh trace start capture=yes report=no maxSize=250 tracefile="%TRACE_FILE%" 2>nul
if %errorlevel% neq 0 (
    echo.
    echo [!] netsh trace falhou. Tenta executar como Administrador.
    echo.
    echo Se quiseres usar so o Metodo 1 (DevTools), isso e suficiente.
    echo.
    pause
    exit /b 1
)

echo.
echo CAPTURA ATIVA! Agora lanca o Arknights Endfield.
echo.
echo Quando o Discord mostrar "Playing Arknights: Endfield",
echo volta aqui e carrega numa tecla para PARAR.
echo.
pause

echo A PARAR captura...
netsh trace stop >nul 2>&1

echo.
echo ✓ Captura guardada em:
echo   %TRACE_FILE%
echo.
echo Para analisar, instala o Wireshark ou o Microsoft Network Monitor.
echo.
echo Mas o MAIS IMPORTANTE e o Metodo 1 (DevTools) - cola o cURL aqui!
echo.
pause
