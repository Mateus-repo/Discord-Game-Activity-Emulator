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

func rnd(a, b int) int {
	if a >= b {
		return a
	}
	return rand.Intn(b-a+1) + a
}

func runVideo(client *DiscordClient, questID string, task TaskInfo) {
	out("  ▶ VIDEO (%.0f/%.0f s)", task.Done, task.Target)
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
		}
		out("  ✓ %.0f/%.0f (%.0f%%)", cur, task.Target, cur/task.Target*100)
		sendProgress(questID, "", string(task.Type), cur, task.Target, "running")
		if cur >= task.Target || term {
			sendProgress(questID, "", string(task.Type), cur, task.Target, "done")
			out("  ✓ VIDEO completo!")
			return
		}
		time.Sleep(time.Duration(rnd(1200, 1800)) * time.Millisecond)
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
	streamKey := fmt.Sprintf("call:%s:%d", ch, rnd(1000, 9999))
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
	streamKey := fmt.Sprintf("call:%s:%d", ch, rnd(1000, 9999))
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
			sendProgress(questID, "", string(task.Type), cur, task.Target, "done")
			out("  ✓ ACHIEVEMENT completo!")
			return
		}
		time.Sleep(time.Duration(rnd(19000, 22000)) * time.Millisecond)
	}
}

func runDesktop(client *DiscordClient, questID string, task TaskInfo) {
	out("  ▶ PLAY_ON_DESKTOP (%.0f/%.0f s)", task.Done, task.Target)
	out("  → A tentar heartbeats com stream_key")

	streamKey := "ineligible_platform"
	cur := task.Done
	fails := 0
	dbg("runDesktop: stream_key=%s", streamKey)

	for i := 0; i < 10; i++ {
		prog, term, err := client.SendHeartbeat(questID, streamKey)
		if err != nil {
			if term {
				out("  ✗ PLAY_ON_DESKTOP não suporta heartbeats diretos.")
				out("  → Alternativa: Executa o jogo real ou usa injeção no cliente Discord.")
				return
			}
			fails++
			dbg("runDesktop: erro (try %d): %v", fails, err)
			if fails >= 3 {
				out("  ✗ PLAY_ON_DESKTOP requer injeção no cliente Discord")
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
		out("  ✓ %.0f/%.0f", cur, task.Target)
		sendProgress(questID, "", string(task.Type), cur, task.Target, "running")
		if cur >= task.Target || term {
			sendProgress(questID, "", string(task.Type), cur, task.Target, "done")
			out("  ✓ PLAY_ON_DESKTOP completo!")
			return
		}
		time.Sleep(time.Duration(rnd(19000, 22000)) * time.Millisecond)
	}
	out("  → PLAY_ON_DESKTOP requer injeção no cliente Discord para funcionar")
}
