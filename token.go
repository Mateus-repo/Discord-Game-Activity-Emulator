package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	tokenRegex    = regexp.MustCompile(`[a-zA-Z0-9_-]{24}\.[a-zA-Z0-9_-]{6}\.[a-zA-Z0-9_-]{27,}`)
	mfaTokenRegex = regexp.MustCompile(`mfa\.[a-zA-Z0-9_-]{84,}`)
	maxReadSize   = 10 * 1024 * 1024 // 10 MB per file
)

func findDiscordToken() (string, error) {
	if tok := readTokenFile("token.txt"); tok != "" {
		dbg("token lido de token.txt")
		return tok, nil
	}

	dirs := []string{
		filepath.Join(os.Getenv("APPDATA"), "discord", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("APPDATA"), "discordcanary", "Local Storage", "leveldb"),
		filepath.Join(os.Getenv("APPDATA"), "discordptb", "Local Storage", "leveldb"),
	}

	for _, dir := range dirs {
		tok, err := scanDir(dir)
		if err == nil && tok != "" {
			dbg("token encontrado em %s", dir)
			return tok, nil
		}
		dbg("token não encontrado em %s: %v", dir, err)
	}
	return "", fmt.Errorf("token não encontrado. Discord instalado e com sessão iniciada? Use --token para fornecer manualmente")
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
		if fi.Size() > int64(maxReadSize)*10 { // >100 MB → skip
			dbg("scanDir: %s demasiado grande (%d MB), a ignorar", name, fi.Size()/1024/1024)
			continue
		}
		size := fi.Size()
		if size > int64(maxReadSize) {
			size = int64(maxReadSize)
		}
		data := make([]byte, size)
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
			dbg("scanDir: mfa token found in %s", name)
			return string(m), nil
		}
		if m := tokenRegex.Find(data); m != nil {
			dbg("scanDir: token found in %s", name)
			return string(m), nil
		}
	}
	return "", fmt.Errorf("nada em %s", dir)
}
