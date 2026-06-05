package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	tokenRegex    = regexp.MustCompile(`[a-zA-Z0-9_-]{24,26}\.[a-zA-Z0-9_-]{6}\.[a-zA-Z0-9_-]{27,}`)
	mfaTokenRegex = regexp.MustCompile(`mfa\.[a-zA-Z0-9_-]{84,}`)
)

const tokenFile = "token.json"

type storedToken struct {
	Token   string `json:"token"`
	Account string `json:"account,omitempty"`
}

func findDiscordToken() (string, error) {
	if tok, _ := readTokenJSON(tokenFile); tok != "" {
		dbg("token lido de %s", tokenFile)
		return tok, nil
	}
	if tok := readTokenFile("token.txt"); tok != "" {
		dbg("token migrado de token.txt para %s", tokenFile)
		return tok, nil
	}

	searchPaths := []string{
		filepath.Join(os.Getenv("APPDATA"), "discord", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("APPDATA"), "discordcanary", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("APPDATA"), "discordptb", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "discord", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "discordcanary", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "discordptb", "Local Storage", "leveldb"),
	}

	for _, dir := range searchPaths {
		tok, err := scanDir(dir)
		if err == nil && tok != "" {
			dbg("token encontrado em %s", dir)
			return tok, nil
		}
		dbg("token não encontrado em %s: %v", dir, err)
	}

	return "", fmt.Errorf("token não encontrado.")
}

func readTokenFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return ""
	}
	if !tokenRegex.MatchString(s) && !mfaTokenRegex.MatchString(s) {
		return ""
	}
	return s
}

func readTokenJSON(path string) (string, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	var st storedToken
	if err := json.Unmarshal(data, &st); err != nil {
		return "", ""
	}
	if st.Token == "" {
		return "", ""
	}
	return st.Token, st.Account
}

func saveToken(tok, account string) {
	if tok == "" {
		return
	}
	existingTok, existingAccount := readTokenJSON(tokenFile)
	if existingTok == tok && existingAccount == account {
		return
	}
	st := storedToken{Token: tok, Account: account}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		dbg("saveToken: json.Marshal: %v", err)
		return
	}
	if err := os.WriteFile(tokenFile, data, 0644); err != nil {
		dbg("saveToken: erro ao guardar: %v", err)
		return
	}
	dbg("token guardado em %s", tokenFile)
}

func scanDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".log") && !strings.HasSuffix(name, ".ldb") {
			continue
		}
		fpath := filepath.Join(dir, name)
		fi, err := os.Stat(fpath)
		if err != nil {
			dbg("scanDir: stat %s: %v", name, err)
			continue
		}
		if fi.Size() > 100*1024*1024 {
			dbg("scanDir: %s demasiado grande (%d MB), a ignorar", name, fi.Size()/1024/1024)
			continue
		}
		data := make([]byte, fi.Size())
		f, err := os.Open(fpath)
		if err != nil {
			dbg("scanDir: open %s: %v", name, err)
			continue
		}
		_, err = f.Read(data)
		f.Close()
		if err != nil {
			dbg("scanDir: read %s: %v", name, err)
			continue
		}
		if m := mfaTokenRegex.Find(data); m != nil {
			return string(m), nil
		}
		if m := tokenRegex.Find(data); m != nil {
			return string(m), nil
		}
	}
	return "", fmt.Errorf("nada em %s", dir)
}
