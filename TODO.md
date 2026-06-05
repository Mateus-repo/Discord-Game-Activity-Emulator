# TODO - Discord Quest Emulator

## Fase 1: Estrutura do Projeto ✅
- [x] Criar repositório e README inicial
- [x] Escolher linguagem (Go: standalone binary, bom suporte HTTP/filesystem)
- [x] Configurar go.mod
- [x] Criar .gitignore
- [x] Criar README.md completo
- [x] Criar TODO.md

## Fase 2: Extração de Token ✅
- [x] Estudar onde o Discord armazena o token (Local Storage / leveldb)
- [x] Implementar leitura do token no Windows (`%APPDATA%/discord/Local Storage/leveldb/`)
- [x] Implementar parser dos ficheiros leveldb para extrair token
- [x] Suporte a fallback (usuário pode fornecer token manualmente com `--token`)
- [x] Validar token (testar com GET /users/@me)

## Fase 3: API Client do Discord ✅
- [x] Implementar cliente HTTP com headers de autenticação
- [x] Rate limiting (retry em 429)
- [x] `GET /users/@me` — verificar token
- [x] `GET /users/@me/quests` — listar quests
- [x] `POST /quests/{id}/enroll` — inscrever em quest
- [x] `POST /quests/{id}/video-progress` — progresso de vídeo
- [x] `POST /quests/{id}/heartbeat` — heartbeat para atividades
- [x] `POST /quests/{id}/claim-reward` — reclamar recompensa
- [x] `GET /applications/public` — obter info de aplicações

## Fase 4: Lógica de Quests ✅
- [x] Modelar tipos de quest (WATCH_VIDEO, PLAY_ACTIVITY, PLAY_ON_DESKTOP, ACHIEVEMENT, STREAM)
- [x] Deteção de tarefas a partir do config da quest (task_config / task_config_v2)
- [x] **WATCH_VIDEO**: enviar timsteps com delay de 1.2-1.8s entre calls
- [x] **PLAY_ACTIVITY**: heartbeats com `call:channelId:random` a cada 19-22s
- [x] **ACHIEVEMENT_IN_ACTIVITY**: heartbeats com fallback a modo manual
- [x] **PLAY_ON_DESKTOP**: tentativa de heartbeats diretos (limitado)
- [x] **STREAM_ON_DESKTOP**: marcado como não suportado
- [x] Auto-enroll antes de processar
- [x] Auto-claim após completar
- [x] PLAY_ON_DESKTOP: implementar heartbeats diretos via API (body: application_id + executable_path + terminal)
- [x] PLAY_ON_DESKTOP: adicionar POST /api/v9/activities para iniciar/parar sessão
- [x] PLAY_ON_DESKTOP: substituir dummy process por heartbeats diretos a cada 20s

## Fase 5: Interface do Utilizador ✅
- [x] CLI básica com flags (--id, --token)
- [x] Listar quests com nomes, tipos, progresso
- [x] Mostrar progresso em tempo real durante execução
- [ ] Opção interativa (selecionar quests)
- [ ] Cor e formatação melhorada

## Fase 6: Build e Distribuição
- [ ] Script de build para Windows (build.bat)
- [x] Testar compilação (go vet ok, build ok)
- [ ] Testar em ambiente real
- [ ] Documentar limitações conhecidas
- [ ] Criar release no GitHub

## Ideias Futuras
- [ ] Suporte a Linux/macOS (ler token de locais diferentes)
- [ ] Injeção de DLL ou processo falso para PLAY_ON_DESKTOP
- [ ] Serviço em background (system tray)
- [ ] TUI interativa com bubbletea
