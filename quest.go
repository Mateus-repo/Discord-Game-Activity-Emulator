package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
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
	exp := time.Now().After(t)
	if exp {
		dbg("isExpired: quest expirada em %s", q.Config.ExpiresAt)
	}
	return exp
}

func runQuests(client *DiscordClient, quests []Quest, channelID string) {
	for _, q := range quests {
		us := q.UserStatus
		if us == nil {
			dbg("runQuests: quest sem user_status, skip")
			continue
		}
		if isExpired(q) {
			fmt.Printf("[!] \"%s\" — expirada\n", questName(q))
			continue
		}
		if us.CompletedAt != "" || us.Completed {
			if us.ClaimedAt != "" {
				fmt.Printf("[✓] \"%s\" — já reclamada\n", questName(q))
			} else {
				fmt.Printf("[✓] \"%s\" — COMPLETA! A reclamar...\n", questName(q))
				claimQuest(client, q)
			}
			continue
		}

		tasks := identifyTasks(q)
		if len(tasks) == 0 {
			fmt.Printf("[?] \"%s\" — sem tarefas reconhecidas\n", questName(q))
			continue
		}

		fmt.Printf("\n▶ \"%s\"\n", questName(q))

		if us.EnrolledAt == "" {
			fmt.Printf("  → A inscrever na quest...\n")
			if err := client.Enroll(q.ID); err != nil {
				fmt.Printf("  ✗ Erro ao inscrever: %v\n", err)
				continue
			}
			fmt.Printf("  ✓ Inscrito!\n")
		}

		for _, t := range tasks {
			if t.Done >= t.Target {
				fmt.Printf("  ✓ %s: já completo (%.0f/%.0f)\n", t.Type, t.Done, t.Target)
				continue
			}
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
	fmt.Printf("  → A reclamar recompensa...\n")
	if err := client.ClaimReward(q.ID); err != nil {
		if strings.Contains(err.Error(), "captcha") || strings.Contains(err.Error(), "confirmado") {
			fmt.Printf("  ⚠ Captcha necessário! Faz claim manualmente no Discord.\n")
		} else {
			fmt.Printf("  ✗ Erro ao reclamar: %v\n", err)
		}
		return
	}
	fmt.Printf("  ✓ Recompensa reclamada!\n")
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
		fmt.Printf("  [?] %s: requer injeção no cliente Discord (modo desktop)\n", task.Type)
	default:
		fmt.Printf("  [?] %s: tipo não implementado\n", task.Type)
	}
}

func rnd(a, b int) int {
	if a >= b {
		return a
	}
	return rand.Intn(b-a+1) + a
}

func runVideo(client *DiscordClient, questID string, task TaskInfo) {
	fmt.Printf("  ▶ VIDEO (%.0f/%.0f s)\n", task.Done, task.Target)
	cur := task.Done
	step := 30.0
	fails := 0
	progressKey := task.Key

	for cur < task.Target {
		sendVal := cur + step
		if sendVal > task.Target {
			sendVal = task.Target
		}
		prog, term, err := client.SendVideoProgress(questID, sendVal, progressKey)
		if err != nil {
			if term {
				fmt.Printf("  ✗ %v\n", err)
				return
			}
			fails++
			dbg("runVideo: erro (try %d): %v", fails, err)
			if fails >= 5 {
				fmt.Printf("  ✗ Demasiados erros, a abortar\n")
				return
			}
			time.Sleep(5 * time.Second)
			continue
		}
		fails = 0
		if prog > cur {
			cur = prog
		}
		pct := cur / task.Target * 100
		fmt.Printf("  ✓ %.0f/%.0f (%.0f%%)\n", cur, task.Target, pct)
		if cur >= task.Target || term {
			fmt.Printf("  ✓ VIDEO completo!\n")
			return
		}
		time.Sleep(time.Duration(rnd(1200, 1800)) * time.Millisecond)
	}
}

func runActivity(client *DiscordClient, questID string, task TaskInfo, channelID string) {
	fmt.Printf("  ▶ ACTIVITY (%.0f/%.0f s)\n", task.Done, task.Target)
	if channelID == "" || channelID == "0" {
		fmt.Printf("  → Sem voice channel. Usa --channel <id> ou entra num voice channel.\n")
	}
	ch := channelID
	if ch == "" {
		ch = "0"
	}
	streamKey := fmt.Sprintf("call:%s:%d", ch, rnd(1000, 9999))
	dbg("runActivity: stream_key=%s", streamKey)
	cur := task.Done
	fails := 0

	for cur < task.Target {
		prog, term, err := client.SendHeartbeat(questID, streamKey)
		if err != nil {
			if term {
				fmt.Printf("  ✗ %v\n", err)
				return
			}
			fails++
			dbg("runActivity: erro (try %d): %v", fails, err)
			if fails >= 5 {
				fmt.Printf("  ✗ Demasiados erros\n")
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
		pct := cur / task.Target * 100
		fmt.Printf("  ✓ %.0f/%.0f (%.0f%%)\n", cur, task.Target, pct)
		if cur >= task.Target || term {
			dbg("runActivity: completo, a enviar terminal heartbeat")
			client.SendHeartbeat(questID, streamKey)
			fmt.Printf("  ✓ ACTIVITY completo!\n")
			return
		}
		time.Sleep(time.Duration(rnd(19000, 22000)) * time.Millisecond)
	}
}

func runAchievement(client *DiscordClient, questID string, task TaskInfo, channelID string) {
	fmt.Printf("  ▶ ACHIEVEMENT (%.0f/%.0f)\n", task.Done, task.Target)
	if channelID == "" || channelID == "0" {
		fmt.Printf("  → Sem voice channel. Usa --channel <id> ou entra num voice channel.\n")
	}
	ch := channelID
	if ch == "" {
		ch = "0"
	}
	streamKey := fmt.Sprintf("call:%s:%d", ch, rnd(1000, 9999))
	dbg("runAchievement: stream_key=%s", streamKey)
	cur := task.Done
	fails := 0

	for cur < task.Target {
		prog, term, err := client.SendHeartbeat(questID, streamKey)
		if err != nil {
			if term {
				fmt.Printf("  ✗ %v\n", err)
				return
			}
			fails++
			dbg("runAchievement: erro (try %d): %v", fails, err)
			if fails >= 5 {
				fmt.Printf("  ✗ ACHIEVEMENT não responde a heartbeats. Completa manualmente.\n")
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
		fmt.Printf("  ✓ progresso: %.0f/%.0f\n", cur, task.Target)
		if cur >= task.Target || term {
			fmt.Printf("  ✓ ACHIEVEMENT completo!\n")
			return
		}
		time.Sleep(time.Duration(rnd(19000, 22000)) * time.Millisecond)
	}
}

func runDesktop(client *DiscordClient, questID string, task TaskInfo) {
	fmt.Printf("  ▶ PLAY_ON_DESKTOP (%.0f/%.0f s)\n", task.Done, task.Target)
	fmt.Printf("  → A tentar heartbeats com stream_key\n")

	streamKey := "ineligible_platform"
	cur := task.Done
	fails := 0
	dbg("runDesktop: stream_key=%s", streamKey)

	for i := 0; i < 10; i++ {
		prog, term, err := client.SendHeartbeat(questID, streamKey)
		if err != nil {
			if term {
				fmt.Printf("  ✗ PLAY_ON_DESKTOP não suporta heartbeats diretos.\n")
				fmt.Printf("  → Alternativa: Executa o jogo real ou usa injeção no cliente Discord.\n")
				return
			}
			fails++
			dbg("runDesktop: erro (try %d): %v", fails, err)
			if fails >= 3 {
				fmt.Printf("  ✗ PLAY_ON_DESKTOP requer injeção no cliente Discord\n")
				return
			}
			time.Sleep(3 * time.Second)
			continue
		}
		fails = 0
		for _, v := range prog {
			if v > cur {
				cur = v
			}
		}
		fmt.Printf("  ✓ %.0f/%.0f\n", cur, task.Target)
		if cur >= task.Target || term {
			fmt.Printf("  ✓ PLAY_ON_DESKTOP completo!\n")
			return
		}
		time.Sleep(time.Duration(rnd(19000, 22000)) * time.Millisecond)
	}
	fmt.Printf("  → PLAY_ON_DESKTOP requer injeção no cliente Discord para funcionar\n")
}
