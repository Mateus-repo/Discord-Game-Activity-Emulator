package main

import (
	"bytes"
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
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Discord/1.0")
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
