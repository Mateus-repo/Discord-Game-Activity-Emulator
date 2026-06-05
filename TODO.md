# TODO - Discord Game Activity Emulator

## Fase 1: Estrutura do Projeto ✅
- [x] Criar repositório e README inicial
- [ ] Escolher linguagem (Go: standalone binary, bom suporte HTTP/filesystem)
- [x] Configurar go.mod
- [ ] Criar estrutura de diretórios

## Fase 2: Extração de Token
- [ ] Estudar onde o Discord armazena o token (Local Storage / leveldb)
- [ ] Implementar leitura do token no Windows (`%APPDATA%/discord/Local Storage/leveldb/`)
- [ ] Implementar parser dos ficheiros leveldb para extrair token
- [ ] Suporte a fallback (usuário pode fornecer token manualmente)
- [ ] Validar token (testar com um GET /users/@me)

## Fase 3: API Client do Discord
- [ ] Implementar cliente HTTP com headers de autenticação
- [ ] `GET /users/@me/quests` — listar quests
- [ ] `POST /quests/{id}/video-progress` — progresso de vídeo
- [ ] `POST /quests/{id}/heartbeat` — heartbeat para atividades
- [ ] `GET /applications/public` — obter info de aplicações das quests
- [ ] Rate limiting / retry handling

## Fase 4: Lógica de Quests
- [ ] Modelar tipos de quest (WATCH_VIDEO, PLAY_ACTIVITY, PLAY_ON_DESKTOP, etc.)
- [ ] Implementar `QuestRunner` interface
- [ ] **WATCH_VIDEO**: enviar timestamps sequenciais até completar
- [ ] **PLAY_ACTIVITY**: gerar stream_key e enviar heartbeats
- [ ] **PLAY_ON_DESKTOP**: pesquisar endpoint correto para heartbeats de jogo
- [ ] **STREAM_ON_DESKTOP**: simular estado de streaming
- [ ] Progress tracking e deteção de conclusão

## Fase 5: Interface do Utilizador
- [ ] Implementar CLI básica (flags: list, run, --quest-id)
- [ ] Mostrar Quests disponíveis com formatação legível
- [ ] Mostrar progresso em tempo real
- [ ] Opção interativa (selecionar quests com setas)

## Fase 6: Build e Distribuição
- [ ] Script de build para Windows (build.bat)
- [ ] Testar em ambiente real
- [ ] Documentar limitações conhecidas
- [ ] Criar release no GitHub

## Ideias Futuras
- [ ] Suporte a Linux/macOS
- [ ] GUI com Tauri/Python
- [ ] Injeção de DLL para suporte total a PLAY_ON_DESKTOP
- [ ] Serviço em background (system tray)
- [ ] Notificações quando quest completar
