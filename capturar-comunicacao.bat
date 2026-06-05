@echo off
title Captura Discord Quest - Arknights Endfield
chcp 65001 >nul
cd /d "%~dp0"

:MENU
cls
echo =============================================
echo    Captura de Comunicacao Discord - Quest
echo    Arknights: Endfield
echo =============================================
echo.
echo Escolhe um metodo:
echo.
echo   [1] Discord DevTools — ve o body dos pedidos (RECOMENDADO)
echo   [2] netsh trace — captura pacotes de rede (alternativa)
echo   [3] Sair
echo.
set /p opcao="Opcao: "

if "%opcao%"=="1" goto DEVTOOLS
if "%opcao%"=="2" goto NETSH
if "%opcao%"=="3" exit /b
goto MENU

:DEVTOOLS
cls
echo =============================================
echo   Metodo 1: Discord DevTools
echo =============================================
echo.
echo Passo a passo:
echo.
echo   PASSO 1 — Abre o Discord (se nao estiver aberto)
echo.
echo   PASSO 2 — Carrega Ctrl+Shift+I para abrir DevTools
echo.
echo   PASSO 3 — Na janela de DevTools, vai a tab "Network"
echo.
echo   PASSO 4 — Escreve "quest" na caixa de filtro (sem aspas)
echo.
echo   PASSO 5 — Mantem o DevTools aberto e lanca o Arknights Endfield
echo             pelo launcher GRYPHLINK
echo.
echo   PASSO 6 — Quando o Discord detetar o jogo, aparecem pedidos
echo             na lista do DevTools (ex: /quests/.../heartbeat)
echo.
echo   PASSO 7 — Clica num pedido com "heartbeat" no nome
echo.
echo   PASSO 8 — Vai a tab "Payload" para ver o corpo do pedido
echo             (stream_key, terminal, etc)
echo.
echo   PASSO 9 — Clica com o direito nesse pedido
echo             -> Copy -> Copy as cURL
echo.
echo   PASSO 10 — Cola o cURL aqui no chat para analisarmos
echo.
echo =============================================
echo.
echo Quando tiveres o cURL, cola aqui no Discord/chat!
echo.
pause
goto MENU

:NETSH
cls
echo =============================================
echo   Metodo 2: netsh trace (captura de rede)
echo =============================================
echo.
echo [!] EXECUTA ESTE .BAT COMO ADMINISTRADOR [!]
echo.
echo Passo a passo:
echo.
echo   PASSO 1 — Fecha o Discord completamente
echo             (clique direito no icone do Discord -> Sair)
echo.
echo   PASSO 2 — Volta a este .bat e carrega numa tecla
echo             para INICIAR a captura
echo.
echo   PASSO 3 — Abre o Discord normalmente
echo
echo   PASSO 4 — Faz login se precisar
echo
echo   PASSO 5 — Espera o Discord carregar completamente
echo
echo   PASSO 6 — Abre o Arknights Endfield pelo launcher GRYPHLINK
echo
echo   PASSO 7 — Espera 30-60 segundos
echo             (ate o Discord mostrar "Playing Arknights: Endfield")
echo
echo   PASSO 8 — Volta a este .bat e carrega numa tecla
echo             para PARAR a captura
echo.
echo   PASSO 9 — O ficheiro .etl fica no Desktop.
echo             Abre com Wireshark ou Microsoft Network Monitor.
echo             Filtra por "http.request.uri contains quest"
echo.
echo =============================================
echo.
pause

echo.
echo A preparar captura...
set TRACE_FILE=%USERPROFILE%\Desktop\discord_quest_%DATE:~-4,4%%DATE:~-7,2%%DATE:~-10,2%_%TIME:~0,2%%TIME:~3,2%%TIME:~6,2%.etl
set TRACE_FILE=%TRACE_FILE: =0%

echo.
echo Ficheiro: %TRACE_FILE%
echo.
echo Carrega numa tecla para INICIAR captura...
echo (Depois abre o Discord e o jogo)
pause

echo.
echo A INICIAR captura...
netsh trace start capture=yes report=no maxSize=250 tracefile="%TRACE_FILE%"
if %errorlevel% neq 0 (
    echo.
    echo [ERRO] netsh trace falhou.
    echo Tenta executar este .bat como Administrador.
    echo.
    pause
    goto MENU
)

echo.
echo ===== CAPTURA ATIVA =====
echo.
echo Agora:
echo   1. Abre o Discord
echo   2. Abre o Arknights Endfield
echo   3. Espera 30-60s
echo.
echo Quando quiseres PARAR, carrega numa tecla.
echo.
pause

echo.
echo A PARAR captura...
netsh trace stop >nul 2>&1

echo.
echo ===== CAPTURA CONCLUIDA =====
echo.
echo Ficheiro: %TRACE_FILE%
echo.
echo Para analisar com Wireshark:
echo   1. Abre o Wireshark
echo   2. File -> Open -> seleciona o .etl
echo   3. Filtra: http.request.uri contains "quest"
echo.
echo Mas tenta primeiro o Metodo 1 (DevTools) - e mais facil!
echo.
pause
goto MENU
