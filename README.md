# Discord Quest Emulator

Aplicação **standalone** (`.exe` único) que completa automaticamente **Quests do Discord** que recompensam **Orbs**, sem precisar instalar ou abrir os jogos reais.

---

## Funcionalidades

- ✅ **WATCH_VIDEO** — envia progresso de visualização direto à API
- ✅ **PLAY_ACTIVITY** — envia heartbeats com `stream_key`
- ✅ **PLAY_ON_DESKTOP** — corre um processo dummy com o nome do jogo para o Discord detetar
- ✅ **Auto-enroll** — inscreve-te automaticamente nas quests
- ✅ **Auto-claim** — reclama a recompensa quando completa
- ✅ **Extração automática do token** — lê dos ficheiros do Discord
- ✅ **CLI simples** — corre tudo ou escolhe uma quest específica
- ❌ **STREAM_ON_DESKTOP** — não suportado (requer injeção no cliente)

## Como funciona

1. Extrai o teu token de autenticação dos ficheiros locais do Discord
2. Consulta `GET /users/@me/quests` para obter as quests ativas
3. Para cada quest:
   - Faz **enroll** se ainda não estiveres inscrito
   - Detecta o tipo de tarefa (WATCH_VIDEO, PLAY_ACTIVITY, PLAY_ON_DESKTOP, etc.)
   - Executa a estratégia adequada:
     - **WATCH_VIDEO**: `POST /quests/{id}/video-progress` com timestamps crescentes
     - **PLAY_ACTIVITY**: `POST /quests/{id}/heartbeat` com `stream_key` `call:channelId:1`
     - **PLAY_ON_DESKTOP**: copia o próprio `.exe` para `games/<appID>/<exeName>.exe`, lança-o em background, e o Discord detecta-o como o jogo real a correr — os heartbeats são enviados automaticamente pelo cliente Discord
4. Quando completa, faz **claim** automático da recompensa

## Pré-requisitos

- Windows (funciona com Discord estável, canary ou PTB)
- Sessão iniciada no Discord pelo menos uma vez
- *(opcional)* Go 1.22+ para compilar manualmente

## Obter o Token Manualmente

Se a extração automática falhar (Discord versões recentes encriptam o token), abre o Discord com **`Ctrl+Shift+I`**, vai à tab **Console** e cola:

```js
(webpackChunkdiscord_app.push([[''],{},e=>{m=[];for(let c in e.c)m.push(e.c[c])}]),m.map(m=>m.exports).filter(x=>x?.default?.getToken?.())[0]?.default?.getToken?.())
```

Copia o output (uma string tipo `OTAyNjA2...`) e guarda num ficheiro `token.txt` ao lado do `.exe`, ou passa com `--token &lt;token&gt;`.

## Uso

```bash
# Corre todas as quests ativas
discord-quest-emulator.exe

# Corre uma quest específica (pelo ID)
discord-quest-emulator.exe --id <quest_id>

# Fornecer token manualmente (se a extração automática falhar)
discord-quest-emulator.exe --token <seu_token>
```

### Exemplo de output

```
=== Discord Quest Emulator ===

✓ Token encontrado
✓ Autenticado como User#1234

Quests encontradas (2):
  • EAFC x The World's Game [WATCH_VIDEO 0/600, PLAY_ON_DESKTOP 0/1800]

A processar quests...

▶ "EAFC x The World's Game"
  → A inscrever na quest...
  ✓ Inscrito!
  ▶ VIDEO (0/600 s)
  ✓ 30/600 (5%)
  ✓ 60/600 (10%)
  ...
  ✓ VIDEO completo!
  ▶ DESKTOP (0/1800 s)
  ✓ A correr processo dummy: games/122131413312612/eafc.exe
  ✓ 15/1800 (1%)
  ✓ 30/1800 (2%)
  ...
  ✓ DESKTOP completo!
  → A reclamar recompensa...
  ✓ Recompensa reclamada!
  ✓ Processo dummy terminado

Concluído! Verifica o progresso no Discord.
```

## Compilar

```bash
git clone <repo>
cd discord-quest-emulator
build.bat
```

## ⚠️ Aviso Legal

Ferramenta apenas para fins educacionais. Automatizar Quests pode violar os ToS do Discord. Usa por tua conta e risco.

## Licença

MIT
