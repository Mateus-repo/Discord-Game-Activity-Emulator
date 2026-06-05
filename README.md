# Discord Game Activity Emulator

Aplicação standalone que simula presença em jogos/atividades no Discord para completar **Quests** (Missões) que recompensam **Orbs**, sem precisar instalar ou abrir os jogos reais.

## Motivação

Completar Quests do Discord normalmente exige instalar, abrir e jogar determinados jogos por um período. Para quem tem internet lenta, SSD com problemas ou simplesmente não quer instalar dezenas de jogos, esta ferramenta automatiza o processo simulando a atividade necessária diretamente via API do Discord.

## Funcionalidades

- **Listar Quests ativas** da sua conta
- **Completar WATCH_VIDEO** — envia progresso de visualização automático
- **Completar PLAY_ACTIVITY** — envia heartbeats simulando participação em atividade
- **Completar PLAY_ON_DESKTOP** — simula deteção de jogo em execução (em desenvolvimento)
- Interface via terminal (CLI/TUI)
- Executável único, sem dependências externas

## Como Funciona

A aplicação:
1. Extrai o token de autenticação do Discord a partir dos ficheiros locais do cliente
2. Consulta a API do Discord (`/users/@me/quests`) para obter as Quests ativas
3. Para cada Quest, determina o tipo de tarefa necessária
4. Executa a estratégia apropriada:
   - **WATCH_VIDEO**: envia `POST /quests/{id}/video-progress` com timestamps incrementais
   - **PLAY_ACTIVITY**: envia `POST /quests/{id}/heartbeat` com `stream_key`
   - **PLAY_ON_DESKTOP**: envia heartbeats simulando a deteção do jogo
5. Acompanha o progresso até a Quest ser concluída

## Pré-requisitos

- **Sistema**: Windows (suporte a Linux/macOS planeado)
- **Discord**: instalado e com sessão iniciada pelo menos uma vez
- **Build**: Go 1.21+ (apenas se quiser compilar manualmente)

## Instalação

### Download do binário (recomendado)
_Futuramente: releases com executáveis pré-compilados._

### Compilar manualmente

```bash
git clone https://github.com/seu-usuario/discord-quest-emulator.git
cd discord-quest-emulator
go build -o discord-quest-emulator.exe .
```

## Uso

```bash
# Mostrar ajuda
discord-quest-emulator.exe --help

# Listar Quests ativas
discord-quest-emulator.exe list

# Completar automaticamente todas as Quests disponíveis
discord-quest-emulator.exe run

# Completar uma Quest específica pelo ID
discord-quest-emulator.exe run --quest-id <id>
```

## Tipos de Quest Suportados

| Tipo | Status | Descrição |
|------|--------|-----------|
| WATCH_VIDEO | ✅ Suportado | Assistir vídeos no Discord |
| PLAY_ACTIVITY | ✅ Suportado | Participar de atividade (voz + atividade) |
| PLAY_ON_DESKTOP | ⚠️ Parcial | Jogar no desktop (requer melhorias) |
| STREAM_ON_DESKTOP | ⚠️ Experimental | Fazer stream no desktop |
| PLAY_ON_DESKTOP_STREAMING | ❌ Não suportado | Stream de jogo específico |
| PLAY_DESKTOP_GAME | ❌ Não suportado | Jogar jogo específico |

## ⚠️ Aviso Legal

Esta ferramenta é fornecida apenas para fins educacionais. O uso de automação para completar Quests pode violar os Termos de Serviço do Discord. Use por sua conta e risco. Não me responsabilizo por quaisquer consequências, incluindo suspensão ou banimento da sua conta.

## Licença

MIT
