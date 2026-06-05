# Discord Quest Emulator

Completa automaticamente **Quests do Discord** que recompensam **Orbs** — sem precisar instalar ou abrir os jogos reais.

Standalone `.exe` (compilação requer CGO/MinGW-w64).

## Funcionalidades

- ✅ **WATCH_VIDEO** — envia progresso de visualização direto à API
- ✅ **PLAY_ACTIVITY** — heartbeats com `stream_key`
- ✅ **PLAY_ON_DESKTOP** — heartbeats diretos via API (com `application_id` + `executable_path`)
- ✅ **ACHIEVEMENT_IN_ACTIVITY** — heartbeat com progresso manual (pode falhar; se após 5 tentativas não funcionar, completa manualmente no Discord)
- ✅ **Auto-enroll** — inscreve-se automaticamente nas quests
- ✅ **Auto-claim** — reclama a recompensa quando completa
- ✅ **Extração automática do token** — lê dos ficheiros do Discord (ou manual com `--token`)
- ✅ **Pronto para qualquer PC** — só copiar o `.exe`, o token é auto-extraído e guardado em `token.json`
- ✅ **Descoberta automática de voice channel** — para quests ACTIVITY/ACHIEVEMENT, encontra um voice channel automaticamente se `--channel` não for fornecido
- ✅ **Deteção de CAPTCHA** — avisa se a recompensa exigir verificação manual
- ❌ **STREAM_ON_DESKTOP** — não suportado (requer injeção no cliente)

## Como funciona

1. Extrai o token de autenticação dos ficheiros locais do Discord (ou usa `--token`)
2. Consulta `GET /quests/@me` para listar as quests ativas
3. Para cada quest faz enroll, processa as tarefas e faz claim automático
4. **PLAY_ON_DESKTOP**: envia heartbeats diretos (`POST /quests/{id}/heartbeat`) com a mesma estrutura que o cliente Discord usa — sem precisar de processos dummy ou jogos instalados

## Uso rápido

```bash
# Corre tudo (lista quests, processa ativas, faz claim)
discord-quest-emulator.exe

# Modo CLI (sem GUI)
discord-quest-emulator.exe --cli

# Apenas uma quest específica
discord-quest-emulator.exe --cli --id <quest_id>

# Fornecer token manualmente
discord-quest-emulator.exe --token <token>

# Verboso (debug)
discord-quest-emulator.exe --cli --debug
```

## Flags

| Flag | Descrição |
|------|-----------|
| `--cli` | Modo terminal (sem GUI) |
| `--id <id>` | Processar apenas uma quest específica |
| `--token <token>` | Fornecer token manualmente |
| `--channel <id>` | ID do voice channel para PLAY_ACTIVITY |
| `--debug` | Logs detalhados |
| `--print-token` | Verificar token (também atualiza `token.json` com o nome da conta) |
| `--stop-desktop <q,a,e>` | Enviar heartbeat terminal para uma quest PLAY_ON_DESKTOP (formato: `questID,appID,exePath`) |
| `--dummy-runner` | Uso interno — iniciado pelo programa para simular processo de jogo |

## Modo Gráfico (GUI)

Por omissão (sem `--cli`), o programa abre uma interface gráfica com:

- **Cartão de autenticação** — mostra se o token foi encontrado ou se é necessário extração manual
- **Lista de quests** — cada quest aparece como um cartão com:
  - Checkbox para selecionar (todas marcadas por defeito)
  - Nome em negrito
  - Ícone do tipo de tarefa: `▶` WATCH_VIDEO, `🖥` PLAY_ON_DESKTOP, `🎮` PLAY_ACTIVITY, `🏆` ACHIEVEMENT
  - Barra de progresso
  - Label de estado colorido: verde "Ativa", azul "Completa", cinza "Reclamada"
- **Botão "Processar selecionadas"** — executa as quests marcadas
- **Botão "Atualizar"** — recarrega a lista de quests
- **Painel de log** — mostra o progresso em tempo real
- **Estado do voice channel** — mostra se foi detetado automaticamente

## Sobre o token

O token é **auto-extraído** dos ficheiros locais do Discord. O programa pesquisa em várias localizações:

| Caminho | Variante |
|---------|----------|
| `%APPDATA%/discord/Local Storage/leveldb/` | Discord Stable |
| `%APPDATA%/discordcanary/Local Storage/leveldb/` | Discord Canary |
| `%APPDATA%/discordptb/Local Storage/leveldb/` | Discord PTB |
| `%LOCALAPPDATA%/discord/Local Storage/leveldb/` | Discord Stable (instalação local) |
| `%LOCALAPPDATA%/discordcanary/Local Storage/leveldb/` | Discord Canary (instalação local) |
| `%LOCALAPPDATA%/discordptb/Local Storage/leveldb/` | Discord PTB (instalação local) |

São lidos ficheiros `.ldb` e `.log` à procura de tokens (incluindo **tokens MFA** com o formato `mfa.xxxx`). Ficheiros com mais de 100 MB são ignorados.

Quando encontrado, o token é guardado em `token.json` (formato `{"token": "...", "account": "..."}`) para usar de imediato nas próximas execuções. O `token.json` é **atualizado automaticamente** em cada execução bem-sucedida com o nome da conta.

Se `token.json` não existir, o programa verifica também um ficheiro `token.txt` legado.

`token.json` está no `.gitignore` — nunca é commitado.

### Obter token manualmente

Se a extração automática falhar, abre o Discord (`Ctrl+Shift+I` > tab **Console**) e experimenta por ordem:

**Método 1 (fetch interceptor — funciona sempre):**
```js
(function() {
  const of = globalThis.fetch;
  globalThis.fetch = async function(...a) {
    const auth = new Headers(a[1]?.headers||{}).get('authorization');
    if (auth) console.log('Token:', auth);
    return of.apply(this, a);
  };
  const oo = XMLHttpRequest.prototype.open;
  const os = XMLHttpRequest.prototype.setRequestHeader;
  XMLHttpRequest.prototype.open = function(m, u, ...r) { this._u = u; return oo.apply(this, [m, u, ...r]); };
  XMLHttpRequest.prototype.setRequestHeader = function(h, v) {
    if (h.toLowerCase() === 'authorization') console.log('Token:', v);
    return os.apply(this, [h, v]);
  };
})();
```
Depois de colares, **clica num DM** para gerar tráfego — o token aparece na Console.

**Método 2 (iframe — se localStorage existir):**
```js
const iframe = document.createElement("iframe");
document.body.appendChild(iframe);
const token = JSON.parse(iframe.contentWindow.localStorage.token);
iframe.remove();
console.log(token);
```

**Método 3 (webpack symbol — pode não funcionar em versões recentes):**
```js
window.webpackChunkdiscord_app.push([[Symbol()],{},o=>{for(let e of Object.values(o.c))try{if(!e.exports||e.exports===window)continue;if(e.exports?.getToken)console.log(e.exports.getToken());for(let o in e.exports)e.exports?.[o]?.getToken&&"IntlMessagesProxy"!==e.exports[o][Symbol.toStringTag]&&console.log(e.exports[o].getToken())}catch{}}]),window.webpackChunkdiscord_app.pop();
```

Guarda o resultado em `token.json`: `{"token": "o-teu-token"}` ou usa `--token <token>`

## Exemplo de output

```
=== Discord Quest Emulator ===

✓ Token encontrado
✓ Autenticado como Strefiz

Quests encontradas (2):
  • Version Update: Mi Fu [PLAY_ON_DESKTOP 122/900, PLAY_ON_DESKTOP 122/900]
  • A Odisseia [ACHIEVEMENT_IN_ACTIVITY 0/1]

▶ "Version Update: Mi Fu"
  ▶ DESKTOP (122/900 s)
  ✓ 142/900 (16%)
  ✓ 162/900 (18%)
  ...
  ✓ DESKTOP completo!
  → A reclamar recompensa...
  ✓ Recompensa reclamada!
```

## Compilar

Requer Go 1.22+ e MinGW-w64 (para CGO — necessário para a GUI Fyne).

```bash
git clone https://github.com/Mateus-repo/Discord-Game-Activity-Emulator.git
cd Discord-Game-Activity-Emulator
build.bat
```

> ⚠️ A primeira compilação pode demorar 5–10 minutos (descarrega dependências e compila a Fyne).

## Script de captura

O `capturar-comunicacao.bat` abre o Discord com debugging para capturar o tráfego HAR — útil se quiseres inspecionar os pedidos que o cliente faz.

## ⚠️ Aviso Legal

Ferramenta apenas para fins educacionais. Automatizar Quests pode violar os ToS do Discord. Usa por tua conta e risco.

## Licença

MIT
