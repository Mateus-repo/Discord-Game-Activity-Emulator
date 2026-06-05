package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://discord.com/api/v9"

type DiscordClient struct {
	token  string
	http   *http.Client
	userID string
	user   string
}

func NewDiscordClient(token string) *DiscordClient {
	return &DiscordClient{
		token: token,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *DiscordClient) do(method, path string, body any) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("json.Marshal: %w", err)
		}
	}
	try := 0
	for {
		try++
		var r io.Reader
		if bodyBytes != nil {
			r = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(method, baseURL+path, r)
		if err != nil {
			return nil, fmt.Errorf("http.NewRequest: %w", err)
		}
		req.Header.Set("Authorization", c.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) discord/1.0.9240 Chrome/138.0.7204.251 Electron/37.6.0 Safari/537.36")
		req.Header.Set("Origin", "https://discord.com")
		req.Header.Set("x-discord-locale", "pt-BR")
		req.Header.Set("x-debug-options", "bugReporterEnabled")
		req.Header.Set("x-super-properties", "eyJvcyI6IldpbmRvd3MiLCJicm93c2VyIjoiRGlzY29yZCBDbGllbnQiLCJyZWxlYXNlX2NoYW5uZWwiOiJzdGFibGUiLCJjbGllbnRfdmVyc2lvbiI6IjEuMC45MjQwIiwib3NfdmVyc2lvbiI6IjEwLjAuMjYyMDAiLCJvc19hcmNoIjoieDY0IiwiYXBwX2FyY2giOiJ4NjQiLCJzeXN0ZW1fbG9jYWxlIjoiZW4tVVMiLCJoYXNfY2xpZW50X21vZHMiOmZhbHNlLCJjbGllbnRfbGF1bmNoX2lkIjoiMTIyNDIwNDctNDFmYy00NGRjLTllZDQtMmI5MTQ3MzI2OTQxIiwiYnJvd3Nlcl91c2VyX2FnZW50IjoiTW96aWxsYS81LjAgKFdpbmRvd3MgTlQgMTAuMDsgV2luNjQ7IHg2NCkgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgZGlzY29yZC8xLjAuOTI0MCBDaHJvbWUvMTM4LjAuNzIwNC4yNTEgRWxlY3Ryb24vMzcuNi4wIFNhZmFyaS81MzcuMzYiLCJicm93c2VyX3ZlcnNpb24iOiIzNy42LjAiLCJvc19zZGtfdmVyc2lvbiI6IjI2MjAwIiwiY2xpZW50X2J1aWxkX251bWJlciI6NTU2OTY5LCJuYXRpdmVfYnVpbGRfbnVtYmVyIjo4MzQzMiwiY2xpZW50X2V2ZW50X3NvdXJjZSI6bnVsbCwibGF1bmNoX3NpZ25hdHVyZSI6ImE1Nzc2ZWM2LTk1NWItNDEyZS05MDU4LWJkMmI5MjhlZTczNyIsImNsaWVudF9oZWFydGJlYXRfc2Vzc2lvbl9pZCI6IjZkMmM3YTYzLTNhZmYtNDU5Zi1hNmJjLTk5NzgwNzk3YjNjMSIsImNsaWVudF9hcHBfc3RhdGUiOiJ1bmZvY3VzZWQifQ==")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http.Do: %w", err)
		}
		if resp.StatusCode == 429 {
			var rb struct {
				RetryAfter float64 `json:"retry_after"`
				Message    string  `json:"message"`
				Global     bool    `json:"global"`
			}
			bodyBytes429, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			json.Unmarshal(bodyBytes429, &rb)
			delay := rb.RetryAfter
			if delay < 1 {
				delay = 5
			}
			dbg("429 rate limit (%s), esperar %.1fs (try %d)", rb.Message, delay, try)
			if try >= 3 {
				return nil, fmt.Errorf("rate limit excedido após %d tentativas", try-1)
			}
			time.Sleep(time.Duration(delay*1000) * time.Millisecond)
			continue
		}
		return resp, nil
	}
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Discrim  string `json:"discriminator"`
	Global   string `json:"global_name"`
}

func (c *DiscordClient) Verify() error {
	dbg("Verify: GET /users/@me")
	resp, err := c.do("GET", "/users/@me", nil)
	if err != nil {
		return fmt.Errorf("erro de rede: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("token inválido ou expirado")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resposta inesperada %d: %s", resp.StatusCode, string(body))
	}
	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return fmt.Errorf("erro a ler resposta: %w", err)
	}
	c.userID = u.ID
	disp := u.Username
	if u.Discrim != "0" {
		disp += "#" + u.Discrim
	} else if u.Global != "" {
		disp = u.Global
	}
	c.user = disp
	dbg("Verify: autenticado como %s (id %s)", disp, u.ID)
	return nil
}

type Quest struct {
	ID            string       `json:"id"`
	ApplicationID string       `json:"application_id"`
	Config        *QuestConfig `json:"config"`
	UserStatus    *UserStatus  `json:"user_status"`
}

type QuestConfig struct {
	Version     int               `json:"config_version"`
	ExpiresAt   string            `json:"expires_at"`
	Application *QuestApplication `json:"application"`
	TaskConfig  json.RawMessage   `json:"task_config"`
	TaskV2      json.RawMessage   `json:"task_config_v2"`
	Rewards     *RewardsConfig    `json:"rewards_config"`
	Messages    *QuestMessages    `json:"messages"`
}

type QuestApplication struct {
	ID string `json:"id"`
}

type RewardsConfig struct {
	Rewards []Reward `json:"rewards"`
}
type Reward struct {
	Type     int            `json:"type"`
	Messages *QuestMessages `json:"messages"`
}

type QuestMessages struct {
	Name string `json:"quest_name"`
}

type UserStatus struct {
	Progress      map[string]ProgressValue `json:"progress"`
	StreamSeconds int                      `json:"stream_progress_seconds"`
	Completed     bool                     `json:"completed"`
	CompletedAt   string                   `json:"completed_at,omitempty"`
	ClaimedAt     string                   `json:"claimed_at,omitempty"`
	EnrolledAt    string                   `json:"enrolled_at,omitempty"`
}

type ProgressValue struct {
	Value float64 `json:"value"`
}

type TasksConfig struct {
	Tasks map[string]TaskDef `json:"tasks"`
}
type TaskDef struct {
	Target int `json:"target"`
}

func (c *DiscordClient) GetQuests() ([]Quest, error) {
	dbg("GetQuests: GET /quests/@me")
	resp, err := c.do("GET", "/quests/@me", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro %d: %s", resp.StatusCode, string(b))
	}
	var ql struct {
		Quests []Quest `json:"quests"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ql); err != nil {
		return nil, fmt.Errorf("json.Decode: %w", err)
	}
	dbg("GetQuests: %d quests encontradas", len(ql.Quests))
	return ql.Quests, nil
}

func (c *DiscordClient) Enroll(questID string) error {
	dbg("Enroll: POST /quests/%s/enroll", questID)
	resp, err := c.do("POST", "/quests/"+questID+"/enroll", map[string]any{
		"location":    11,
		"is_targeted": false,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("enroll erro %d: %s", resp.StatusCode, string(b))
	}
	dbg("Enroll: sucesso")
	return nil
}

func (c *DiscordClient) ClaimReward(questID string) error {
	dbg("ClaimReward: POST /quests/%s/claim-reward", questID)
	resp, err := c.do("POST", "/quests/"+questID+"/claim-reward", map[string]any{
		"platform":                0,
		"location":                11,
		"is_targeted":             false,
		"metadata_raw":            nil,
		"metadata_sealed":         nil,
		"traffic_metadata_raw":    nil,
		"traffic_metadata_sealed": nil,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("claim erro %d: %s", resp.StatusCode, string(b))
	}
	var result struct {
		ClaimedAt string `json:"claimed_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("json.Decode: %w", err)
	}
	if result.ClaimedAt == "" {
		return fmt.Errorf("claim não confirmado (captcha?)")
	}
	dbg("ClaimReward: sucesso, claimed_at=%s", result.ClaimedAt)
	return nil
}

func (c *DiscordClient) SendVideoProgress(questID string, timestamp float64, progressKey string) (float64, bool, error) {
	dbg("SendVideoProgress: quest=%s ts=%.0f key=%s", questID, timestamp, progressKey)
	resp, err := c.do("POST", "/quests/"+questID+"/video-progress", map[string]float64{
		"timestamp": timestamp,
	})
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 {
		return 0, true, fmt.Errorf("quest inválida ou expirada")
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return 0, false, fmt.Errorf("erro %d: %s", resp.StatusCode, string(b))
	}
	var vpr struct {
		Progress    map[string]ProgressValue `json:"progress"`
		CompletedAt string                   `json:"completed_at,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vpr); err != nil {
		return 0, false, fmt.Errorf("json.Decode: %w", err)
	}
	if progressKey != "" {
		if pv, ok := vpr.Progress[progressKey]; ok {
			return pv.Value, vpr.CompletedAt != "", nil
		}
	}
	for _, pv := range vpr.Progress {
		return pv.Value, vpr.CompletedAt != "", nil
	}
	return 0, vpr.CompletedAt != "", nil
}

func (c *DiscordClient) SendHeartbeat(questID, streamKey string) (map[string]float64, bool, error) {
	dbg("SendHeartbeat: quest=%s stream_key=%s", questID, streamKey)
	resp, err := c.do("POST", "/quests/"+questID+"/heartbeat", map[string]any{
		"stream_key": streamKey,
		"terminal":   false,
	})
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 {
		return nil, true, fmt.Errorf("quest inválida ou expirada")
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("erro %d: %s", resp.StatusCode, string(b))
	}
	var hr struct {
		Progress    map[string]ProgressValue `json:"progress"`
		CompletedAt string                   `json:"completed_at,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, false, fmt.Errorf("json.Decode: %w", err)
	}
	out := make(map[string]float64)
	for k, v := range hr.Progress {
		out[k] = v.Value
	}
	dbg("SendHeartbeat: progress=%v completed=%v", out, hr.CompletedAt != "")
	return out, hr.CompletedAt != "", nil
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *DiscordClient) SendDesktopHeartbeat(questID, appID, exePath string, terminal bool) (map[string]float64, bool, error) {
	body := map[string]any{
		"terminal": terminal,
	}
	if !terminal {
		body["application_id"] = appID
		body["executable_path"] = exePath
	}
	dbg("SendDesktopHeartbeat: quest=%s terminal=%v body=%v", questID, terminal, body)
	resp, err := c.do("POST", "/quests/"+questID+"/heartbeat", body)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 {
		return nil, true, fmt.Errorf("quest inválida ou expirada")
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("erro %d: %s", resp.StatusCode, string(b))
	}
	var hr struct {
		Progress    map[string]ProgressValue `json:"progress"`
		CompletedAt string                   `json:"completed_at,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, false, fmt.Errorf("json.Decode: %w", err)
	}
	out := make(map[string]float64)
	for k, v := range hr.Progress {
		out[k] = v.Value
	}
	dbg("SendDesktopHeartbeat: progress=%v completed=%v", out, hr.CompletedAt != "")
	return out, hr.CompletedAt != "", nil
}

func (c *DiscordClient) CallActivity(appID, exePath, sessionID, token string, duration int, closed bool) (string, error) {
	var tok *string
	if token != "" {
		tok = &token
	}
	body := map[string]any{
		"application_id":   appID,
		"token":            tok,
		"duration":         duration,
		"share_activity":   true,
		"closed":           closed,
		"exePath":          exePath,
		"voice_channel_id": nil,
		"session_id":       sessionID,
		"media_session_id": nil,
	}
	dbg("CallActivity: app=%s closed=%v", appID, closed)
	resp, err := c.do("POST", "/activities", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("activity erro %d: %s", resp.StatusCode, string(b))
	}
	var ar struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return "", fmt.Errorf("json.Decode: %w", err)
	}
	dbg("CallActivity: token=%s", ar.Token[:min(len(ar.Token), 20)])
	return ar.Token, nil
}

func (c *DiscordClient) StartActivity(appID, exePath, sessionID string) (string, error) {
	return c.CallActivity(appID, exePath, sessionID, "", 0, false)
}

func (c *DiscordClient) StopActivity(appID, exePath, sessionID, token string, duration int) error {
	_, err := c.CallActivity(appID, exePath, sessionID, token, duration, true)
	return err
}

type Guild struct {
	ID string `json:"id"`
}
type Channel struct {
	ID   string `json:"id"`
	Type int    `json:"type"`
	Name string `json:"name"`
}

func (c *DiscordClient) GetGuilds() ([]Guild, error) {
	dbg("GetGuilds: GET /users/@me/guilds")
	resp, err := c.do("GET", "/users/@me/guilds", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro %d", resp.StatusCode)
	}
	var gs []Guild
	if err := json.NewDecoder(resp.Body).Decode(&gs); err != nil {
		return nil, err
	}
	dbg("GetGuilds: %d guilds", len(gs))
	return gs, nil
}

func (c *DiscordClient) GetChannels(guildID string) ([]Channel, error) {
	dbg("GetChannels: GET /guilds/%s/channels", guildID)
	resp, err := c.do("GET", "/guilds/"+guildID+"/channels", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro %d", resp.StatusCode)
	}
	var cs []Channel
	if err := json.NewDecoder(resp.Body).Decode(&cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (c *DiscordClient) FindVoiceChannel() string {
	dbg("FindVoiceChannel: a procurar voice channels...")
	guilds, err := c.GetGuilds()
	if err != nil {
		dbg("FindVoiceChannel: erro guilds: %v", err)
		return ""
	}
	for _, g := range guilds {
		channels, err := c.GetChannels(g.ID)
		if err != nil {
			dbg("FindVoiceChannel: erro channels guild %s: %v", g.ID, err)
			continue
		}
		for _, ch := range channels {
			if ch.Type == 2 {
				dbg("FindVoiceChannel: encontrado voice channel %s (%s)", ch.ID, ch.Name)
				return ch.ID
			}
		}
	}
	dbg("FindVoiceChannel: nenhum voice channel encontrado")
	return ""
}

func (c *DiscordClient) GetDetectableGames() ([]map[string]any, error) {
	dbg("GetDetectableGames: GET /applications/detectable")
	resp, err := c.do("GET", "/applications/detectable", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro %d: %s", resp.StatusCode, string(b))
	}
	var games []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, err
	}
	dbg("GetDetectableGames: %d jogos reconhecidos", len(games))
	return games, nil
}

func findWin32Exe(info map[string]any) string {
	if exes, ok := info["executables"].([]any); ok {
		for _, e := range exes {
			if em, ok := e.(map[string]any); ok {
				if os, ok := em["os"].(string); ok && os == "win32" {
					if name, ok := em["name"].(string); ok {
						return name
					}
				}
			}
		}
	}
	return ""
}

func (c *DiscordClient) GetAppInfo(appID string) (map[string]any, error) {
	dbg("GetAppInfo: app_id=%s", appID)
	resp, err := c.do("GET", "/applications/public?application_ids="+appID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro %d", resp.StatusCode)
	}
	var result []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("app não encontrada")
	}
	return result[0], nil
}
