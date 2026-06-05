package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type QuestType string

const (
	WatchVideo    QuestType = "WATCH_VIDEO"
	PlayOnDesktop QuestType = "PLAY_ON_DESKTOP"
	PlayActivity  QuestType = "PLAY_ACTIVITY"
	StreamDesktop QuestType = "STREAM_ON_DESKTOP"
	Achievement   QuestType = "ACHIEVEMENT_IN_ACTIVITY"
	Unknown       QuestType = "UNKNOWN"
)

type TaskInfo struct {
	Type   QuestType
	Key    string
	Target float64
	Done   float64
	AppID  string
}

const (
	VIDEO_TARGET       = 600.0
	ACTIVITY_TARGET    = 900.0
	DESKTOP_TARGET     = 1800.0
	STREAM_TARGET      = 1800.0
	ACHIEVEMENT_TARGET = 1.0
)

func identifyTasks(q Quest) []TaskInfo {
	dbg("identifyTasks: quest=%s", questName(q))
	var tasks []TaskInfo
	us := q.UserStatus
	if us == nil {
		dbg("identifyTasks: user_status nil")
		return tasks
	}

	dbg("identifyTasks: progress keys: %v", mapKeys(us.Progress))
	cfg := getTaskConfig(q.Config)
	if cfg == nil {
		dbg("identifyTasks: sem task_config, a inferir de progress keys")
		for k, pv := range us.Progress {
			t := inferType(k)
			dbg("identifyTasks:  inferred %s from key %s (done=%.0f)", t, k, pv.Value)
			tasks = append(tasks, TaskInfo{
				Type:   t,
				Key:    k,
				Target: targetForType(t),
				Done:   pv.Value,
				AppID:  appID(q),
			})
		}
		if us.StreamSeconds > 0 {
			dbg("identifyTasks: stream_seconds=%d", us.StreamSeconds)
			tasks = append(tasks, TaskInfo{
				Type:   WatchVideo,
				Key:    "stream",
				Target: VIDEO_TARGET,
				Done:   float64(us.StreamSeconds),
			})
		}
		return tasks
	}

	dbg("identifyTasks: task_config tasks: %v", taskKeys(cfg.Tasks))
	for key, def := range cfg.Tasks {
		d := float64(def.Target)
		t := inferType(key)
		var done float64 = 0
		if pv, ok := us.Progress[key]; ok {
			done = pv.Value
			dbg("identifyTasks:  key=%s type=%s done=%.0f target=%.0f (from exact key)", key, t, done, d)
		} else if pv, ok := us.Progress[string(t)]; ok {
			done = pv.Value
			dbg("identifyTasks:  key=%s type=%s done=%.0f target=%.0f (from type key)", key, t, done, d)
		} else {
			dbg("identifyTasks:  key=%s type=%s done=0 target=%.0f (no progress yet)", key, t, d)
		}
		tasks = append(tasks, TaskInfo{
			Type:   t,
			Key:    key,
			Target: d,
			Done:   done,
			AppID:  appID(q),
		})
	}
	return tasks
}

func mapKeys(m map[string]ProgressValue) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func taskKeys(m map[string]TaskDef) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func getTaskConfig(cfg *QuestConfig) *TasksConfig {
	if cfg == nil {
		return nil
	}
	var tc TasksConfig
	if cfg.TaskV2 != nil {
		if err := tryUnmarshal(cfg.TaskV2, &tc); err == nil && len(tc.Tasks) > 0 {
			dbg("getTaskConfig: loaded from task_config_v2 (%d tasks)", len(tc.Tasks))
			return &tc
		}
	}
	if cfg.TaskConfig != nil {
		if err := tryUnmarshal(cfg.TaskConfig, &tc); err == nil && len(tc.Tasks) > 0 {
			dbg("getTaskConfig: loaded from task_config (%d tasks)", len(tc.Tasks))
			return &tc
		}
	}
	return nil
}

func tryUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func inferType(key string) QuestType {
	upper := strings.ToUpper(key)
	if strings.Contains(upper, "VIDEO") {
		return WatchVideo
	}
	if strings.Contains(upper, "STREAM") {
		return StreamDesktop
	}
	if strings.Contains(upper, "ACHIEVEMENT") {
		return Achievement
	}
	if strings.Contains(upper, "ACTIVITY") {
		return PlayActivity
	}
	if strings.Contains(upper, "PLAY") || strings.Contains(upper, "GAME") || strings.Contains(upper, "DESKTOP") {
		return PlayOnDesktop
	}
	dbg("inferType: unknown key %s", key)
	return Unknown
}

func targetForType(t QuestType) float64 {
	switch t {
	case WatchVideo:
		return VIDEO_TARGET
	case PlayOnDesktop:
		return DESKTOP_TARGET
	case PlayActivity:
		return ACTIVITY_TARGET
	case StreamDesktop:
		return STREAM_TARGET
	case Achievement:
		return ACHIEVEMENT_TARGET
	default:
		return 600
	}
}

func appID(q Quest) string {
	if q.ApplicationID != "" {
		return q.ApplicationID
	}
	if q.Config != nil && q.Config.Application != nil && q.Config.Application.ID != "" {
		return q.Config.Application.ID
	}
	return ""
}

func questName(q Quest) string {
	if q.Config != nil && q.Config.Messages != nil && q.Config.Messages.Name != "" {
		return q.Config.Messages.Name
	}
	if q.ID != "" {
		if len(q.ID) > 10 {
			return q.ID[:10] + "..."
		}
		return q.ID
	}
	return "Unknown"
}

func isExpired(q Quest) bool {
	if q.Config == nil || q.Config.ExpiresAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, q.Config.ExpiresAt)
	if err != nil {
		dbg("isExpired: parse error %s: %v", q.Config.ExpiresAt, err)
		return false
	}
	return time.Now().After(t)
}

func runQuests(client *DiscordClient, quests []Quest, channelID string) {
	for _, q := range quests {
		us := q.UserStatus
		if us == nil {
			dbg("runQuests: quest sem user_status, skip")
			continue
		}
		if isExpired(q) {
			out("[!] \"%s\" — expirada", questName(q))
			continue
		}
		if us.CompletedAt != "" || us.Completed {
			if us.ClaimedAt != "" {
				out("[✓] \"%s\" — já reclamada", questName(q))
			} else {
				out("[✓] \"%s\" — COMPLETA! A reclamar...", questName(q))
				claimQuest(client, q)
			}
			continue
		}

		tasks := identifyTasks(q)
		if len(tasks) == 0 {
			out("[?] \"%s\" — sem tarefas reconhecidas", questName(q))
			continue
		}

		qn := questName(q)
		out("")
		out("▶ \"%s\"", qn)

		if us.EnrolledAt == "" {
			out("  → A inscrever na quest...")
			if err := client.Enroll(q.ID); err != nil {
				out("  ✗ Erro ao inscrever: %v", err)
				continue
			}
			out("  ✓ Inscrito!")
		}

		for _, t := range tasks {
			if t.Done >= t.Target {
				out("  ✓ %s: já completo (%.0f/%.0f)", t.Type, t.Done, t.Target)
				sendProgress(q.ID, qn, string(t.Type), t.Done, t.Target, "done")
				continue
			}
			sendProgress(q.ID, qn, string(t.Type), t.Done, t.Target, "running")
			runTask(client, q.ID, t, channelID)
		}

		dbg("runQuests: a verificar se quest %s foi completada", q.ID)
		if isCompleted(client, q.ID) {
			claimQuest(client, q)
		}
	}
}

func isCompleted(client *DiscordClient, questID string) bool {
	qs, err := client.GetQuests()
	if err != nil {
		dbg("isCompleted: erro GetQuests: %v", err)
		return false
	}
	for _, q := range qs {
		if q.ID == questID && q.UserStatus != nil {
			done := q.UserStatus.Completed || q.UserStatus.CompletedAt != ""
			dbg("isCompleted: quest=%s completed=%v", questID, done)
			return done
		}
	}
	return false
}

func claimQuest(client *DiscordClient, q Quest) {
	out("  → A reclamar recompensa...")
	if err := client.ClaimReward(q.ID); err != nil {
		if strings.Contains(err.Error(), "captcha") || strings.Contains(err.Error(), "confirmado") {
			out("  ⚠ Captcha necessário! Faz claim manualmente no Discord.")
		} else {
			out("  ✗ Erro ao reclamar: %v", err)
		}
		return
	}
	out("  ✓ Recompensa reclamada!")
}

func runTask(client *DiscordClient, questID string, task TaskInfo, channelID string) {
	dbg("runTask: quest=%s type=%s key=%s target=%.0f done=%.0f", questID, task.Type, task.Key, task.Target, task.Done)
	switch task.Type {
	case WatchVideo:
		runVideo(client, questID, task)
	case PlayActivity:
		runActivity(client, questID, task, channelID)
	case Achievement:
		runAchievement(client, questID, task, channelID)
	case PlayOnDesktop:
		runDesktop(client, questID, task)
	case StreamDesktop:
		out("  [?] %s: requer injeção no cliente Discord (modo desktop)", task.Type)
	default:
		out("  [?] %s: tipo não implementado", task.Type)
	}
}

func startDummyGame(exeName, appID string) (*exec.Cmd, string, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return nil, "", fmt.Errorf("os.Executable: %w", err)
	}

	gameDir := filepath.Join("games", appID)
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		return nil, "", fmt.Errorf("mkdir %s: %w", gameDir, err)
	}

	gameExePath := filepath.Join(gameDir, exeName)
	data, err := os.ReadFile(selfPath)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", selfPath, err)
	}
	if err := os.WriteFile(gameExePath, data, 0755); err != nil {
		return nil, "", fmt.Errorf("write %s: %w", gameExePath, err)
	}

	cmd := exec.Command(gameExePath, "--dummy-runner")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	if err := cmd.Start(); err != nil {
		os.Remove(gameExePath)
		return nil, "", fmt.Errorf("start %s: %w", gameExePath, err)
	}

	return cmd, gameExePath, nil
}

func stopDummyGame(cmd *exec.Cmd, gameExePath string) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}
	if gameExePath != "" {
		os.RemoveAll(filepath.Dir(gameExePath))
	}
}

func rnd(a, b int) int {
	if a >= b {
		return a
	}
	return rand.Intn(b-a+1) + a
}

func runVideo(client *DiscordClient, questID string, task TaskInfo) {
	out("  ▶ VIDEO (%.0f/%.0f s)", task.Done, task.Target)
	cur := task.Done
	fails := 0
	progressKey := task.Key
	started := time.Now()

	for cur < task.Target {
		elapsed := time.Since(started).Seconds()
		maxAllowed := elapsed + 10
		speed := 7.0
		timestamp := cur + speed
		if timestamp > maxAllowed {
			timestamp = maxAllowed
		}
		if timestamp > task.Target {
			timestamp = task.Target
		}
		prog, term, err := client.SendVideoProgress(questID, timestamp, progressKey)
		if err != nil {
			if term {
				out("  ✗ %v", err)
				return
			}
			fails++
			dbg("runVideo: erro (try %d): %v", fails, err)
			if fails >= 5 {
				out("  ✗ Demasiados erros, a abortar")
				return
			}
			time.Sleep(5 * time.Second)
			continue
		}
		fails = 0
		if prog > cur {
			cur = prog
		} else if timestamp > cur {
			cur = timestamp
		}
		out("  ✓ %.0f/%.0f (%.0f%%)", cur, task.Target, cur/task.Target*100)
		sendProgress(questID, "", string(task.Type), cur, task.Target, "running")
		if cur >= task.Target || term {
			sendProgress(questID, "", string(task.Type), cur, task.Target, "done")
			out("  ✓ VIDEO completo!")
			return
		}
		time.Sleep(time.Duration(rnd(7000, 9500)) * time.Millisecond)
	}
}

func runActivity(client *DiscordClient, questID string, task TaskInfo, channelID string) {
	out("  ▶ ACTIVITY (%.0f/%.0f s)", task.Done, task.Target)
	if channelID == "" || channelID == "0" {
		out("  → Sem voice channel. Usa --channel <id> ou entra num voice channel.")
	}
	ch := channelID
	if ch == "" {
		ch = "0"
	}
	streamKey := fmt.Sprintf("call:%s:1", ch)
	dbg("runActivity: stream_key=%s", streamKey)
	cur := task.Done
	fails := 0

	for cur < task.Target {
		prog, term, err := client.SendHeartbeat(questID, streamKey)
		if err != nil {
			if term {
				out("  ✗ %v", err)
				return
			}
			fails++
			dbg("runActivity: erro (try %d): %v", fails, err)
			if fails >= 5 {
				out("  ✗ Demasiados erros")
				return
			}
			time.Sleep(5 * time.Second)
			continue
		}
		fails = 0

		hasProg := false
		for _, v := range prog {
			if v > cur {
				cur = v
			}
			hasProg = true
		}
		if !hasProg {
			cur += 20
			dbg("runActivity: progress map vazio, a incrementar +20")
		}
		out("  ✓ %.0f/%.0f (%.0f%%)", cur, task.Target, cur/task.Target*100)
		sendProgress(questID, "", string(task.Type), cur, task.Target, "running")
		if cur >= task.Target || term {
			dbg("runActivity: completo, a enviar terminal heartbeat")
			client.SendHeartbeat(questID, streamKey)
			sendProgress(questID, "", string(task.Type), cur, task.Target, "done")
			out("  ✓ ACTIVITY completo!")
			return
		}
		time.Sleep(time.Duration(rnd(19000, 22000)) * time.Millisecond)
	}
}

func runAchievement(client *DiscordClient, questID string, task TaskInfo, channelID string) {
	out("  ▶ ACHIEVEMENT (%.0f/%.0f)", task.Done, task.Target)
	if channelID == "" || channelID == "0" {
		out("  → Sem voice channel. Usa --channel <id> ou entra num voice channel.")
	}
	ch := channelID
	if ch == "" {
		ch = "0"
	}
	streamKey := fmt.Sprintf("call:%s:1", ch)
	dbg("runAchievement: stream_key=%s", streamKey)
	cur := task.Done
	fails := 0

	for cur < task.Target {
		prog, term, err := client.SendHeartbeat(questID, streamKey)
		if err != nil {
			if term {
				out("  ✗ %v", err)
				return
			}
			fails++
			dbg("runAchievement: erro (try %d): %v", fails, err)
			if fails >= 5 {
				out("  ✗ ACHIEVEMENT não responde a heartbeats. Completa manualmente.")
				return
			}
			time.Sleep(5 * time.Second)
			continue
		}
		fails = 0
		for _, v := range prog {
			if v > cur {
				cur = v
			}
		}
		out("  ✓ progresso: %.0f/%.0f", cur, task.Target)
		sendProgress(questID, "", string(task.Type), cur, task.Target, "running")
		if cur >= task.Target || term {
			dbg("runAchievement: completo, a enviar terminal heartbeat")
			client.SendHeartbeat(questID, streamKey)
			sendProgress(questID, "", string(task.Type), cur, task.Target, "done")
			out("  ✓ ACHIEVEMENT completo!")
			return
		}
		time.Sleep(time.Duration(rnd(19000, 22000)) * time.Millisecond)
	}
}

func runDesktop(client *DiscordClient, questID string, task TaskInfo) {
	out("  ▶ DESKTOP (%.0f/%.0f s)", task.Done, task.Target)

	appID := task.AppID

	var exeName, appName string
	if appID != "" {
		info, err := client.GetAppInfo(appID)
		if err == nil {
			if name, ok := info["name"].(string); ok {
				appName = name
			}
			exeName = findWin32Exe(info)
			if exeName != "" {
				out("  → Jogo: %s | EXE: %s", appName, exeName)
			} else {
				out("  → Jogo: %s (ID: %s)", appName, appID)
			}
		}
	}

	if exeName == "" {
		out("  → A pesquisar na lista de jogos detetáveis...")
		games, err := client.GetDetectableGames()
		if err == nil {
			for _, g := range games {
				gid, _ := g["id"].(string)
				if gid == appID {
					if name, ok := g["name"].(string); ok {
						appName = name
					}
					exeName = findWin32Exe(g)
					if exeName != "" {
						out("  → Encontrado na lista: %s | EXE: %s", appName, exeName)
					}
					break
				}
			}
		}
	}

	if exeName == "" {
		out("  ✗ Não foi possível determinar o executável do jogo.")
		out("  → Alternativa 1: Abre Discord (Ctrl+Shift+I) > Console e cola o script de outros-scripts/script-1-funciona.txt")
		out("  → Alternativa 2: Executa o jogo real com o Discord aberto")
		return
	}

	cleanName := strings.ReplaceAll(exeName, ">", "")

	cmd, gamePath, err := startDummyGame(cleanName, appID)
	if err != nil {
		out("  ✗ Erro ao iniciar processo dummy: %v", err)
		return
	}
	defer stopDummyGame(cmd, gamePath)
	out("  ✓ Processo dummy: %s", gamePath)

	for {
		qs, err := client.GetQuests()
		if err != nil {
			dbg("runDesktop: GetQuests error: %v", err)
			time.Sleep(15 * time.Second)
			continue
		}

		var qsUs *UserStatus
		for _, q := range qs {
			if q.ID == questID {
				qsUs = q.UserStatus
				break
			}
		}
		if qsUs == nil {
			out("  ✗ Quest não encontrada na lista")
			return
		}

		cur := float64(0)
		if task.Key != "" {
			if pv, ok := qsUs.Progress[task.Key]; ok {
				cur = pv.Value
			}
		}
		for _, pv := range qsUs.Progress {
			if pv.Value > cur {
				cur = pv.Value
			}
		}

		out("  ✓ %.0f/%.0f (%.0f%%)", cur, task.Target, cur/task.Target*100)
		sendProgress(questID, "", string(task.Type), cur, task.Target, "running")

		if qsUs.Completed || qsUs.CompletedAt != "" || cur >= task.Target {
			dbg("runDesktop: quest completa!")
			sendProgress(questID, "", string(task.Type), cur, task.Target, "done")
			out("  ✓ DESKTOP completo!")
			return
		}

		time.Sleep(20 * time.Second)
	}
}
