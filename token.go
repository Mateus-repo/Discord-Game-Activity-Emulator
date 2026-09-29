package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	tokenRegex    = regexp.MustCompile(`[a-zA-Z0-9_-]{24,26}\.[a-zA-Z0-9_-]{6}\.[a-zA-Z0-9_-]{27,}`)
	mfaTokenRegex = regexp.MustCompile(`mfa\.[a-zA-Z0-9_-]{84,}`)
)

const (
	tokenFile   = "token.json"
	legacyFile  = "token.txt"
	tokenFormat = `[primeira-parte].[timestamp].[assinatura] (ex: mta.xxxxx.aaaaaaa) ou mfa.xxxxx`
)

type storedToken struct {
	Token   string `json:"token"`
	Account string `json:"account,omitempty"`
}

// findDiscordToken procura o token em:
//  1. token.json  (formato {"token": "...", "account": "..."})
//  2. token.txt   (texto simples com o token)
//  3. %APPDATA% / %LOCALAPPDATA%\discord*\Local Storage\leveldb
//
// Os ficheiros são procurados tanto na pasta de trabalho atual como na pasta
// do executável, para funcionar mesmo se o .exe for aberto duma pasta diferente.
func findDiscordToken() (string, error) {
	var problems []string

	dirs := tokenSearchDirs()

	for _, dir := range dirs {
		jsonPath := filepath.Join(dir, tokenFile)
		if data, err := os.ReadFile(jsonPath); err == nil {
			tok, reason := tokenFromJSON(data)
			if tok != "" {
				dbg("token lido de %s", jsonPath)
				return tok, nil
			}
			problems = append(problems, fmt.Sprintf("%s existe mas %s", jsonPath, reason))
		}

		txtPath := filepath.Join(dir, legacyFile)
		if data, err := os.ReadFile(txtPath); err == nil {
			tok, reason := tokenFromText(string(data))
			if tok != "" {
				dbg("token lido de %s", txtPath)
				return tok, nil
			}
			problems = append(problems, fmt.Sprintf("%s existe mas %s", txtPath, reason))
		}
	}

	for _, dir := range leveldbDirs() {
		tok, err := scanDir(dir)
		if err == nil && tok != "" {
			dbg("token encontrado em %s", dir)
			return tok, nil
		}
	}

	if len(problems) == 0 {
		return "", fmt.Errorf("token não encontrado. Nenhum token.json/token.txt válido encontrado.")
	}
	return "", fmt.Errorf("token não encontrado:\n  - %s\n\nFormato esperado: %s",
		strings.Join(problems, "\n  - "), tokenFormat)
}

// tokenSearchDirs devolve as pastas onde procurar token.json / token.txt,
// sem duplicados. A pasta do executável vem em último para não sobrepor o que
// o utilizador está a editar, mas é sempre incluída.
func tokenSearchDirs() []string {
	dirs := []string{"."}

	if self, err := os.Executable(); err == nil {
		if exeDir := filepath.Dir(self); exeDir != "" {
			dirs = append(dirs, exeDir)
		}
	}

	seen := make(map[string]bool, len(dirs))
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		key := strings.ToLower(abs)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, abs)
	}
	return out
}

func leveldbDirs() []string {
	roots := []struct{ env, name string }{
		{"APPDATA", "discord"},
		{"APPDATA", "discordcanary"},
		{"APPDATA", "discordptb"},
		{"LOCALAPPDATA", "discord"},
		{"LOCALAPPDATA", "discordcanary"},
		{"LOCALAPPDATA", "discordptb"},
	}
	var dirs []string
	for _, r := range roots {
		base := os.Getenv(r.env)
		if base == "" {
			continue
		}
		dirs = append(dirs, filepath.Join(base, r.name, "Local Storage", "leveldb"))
	}
	return dirs
}

// tokenFromText extrai um token de texto livre. Tolera um token.txt gravado em
// UTF-16 (Notepad), com BOM, com aspas, com "Bearer" à frente ou com o token
// colado no meio de outras linhas.
func tokenFromText(content string) (string, string) {
	cleaned := cleanTokenText(content)
	if cleaned == "" {
		return "", "está vazio"
	}
	if m := mfaTokenRegex.FindString(cleaned); m != "" {
		return m, ""
	}
	if m := tokenRegex.FindString(cleaned); m != "" {
		return m, ""
	}
	return "", fmt.Sprintf("o conteúdo não parece um token do Discord (esperado: %s)", tokenFormat)
}

func tokenFromJSON(data []byte) (string, string) {
	var st storedToken
	if err := json.Unmarshal([]byte(deBOM(string(data))), &st); err != nil {
		return "", `o JSON é inválido (esperado: {"token": "..."})`
	}
	if strings.TrimSpace(st.Token) == "" {
		return "", `o campo "token" está vazio`
	}
	if tok, _ := tokenFromText(st.Token); tok != "" {
		return tok, ""
	}
	return "", fmt.Sprintf("o campo \"token\" não parece um token do Discord (esperado: %s)", tokenFormat)
}

// deBOM remove BOM UTF-8/UTF-16 e bytes NUL (ficheiros guardados em UTF-16,
// que é o default antigo do Notepad). Um token nunca contém NUL, por isso
// removê-los é seguro e evita que o token fique partido por bytes inválidos.
//
// A remoção do BOM é feita sobre os bytes crus: um BOM UTF-16 (FF FE) não é
// UTF-8 válido, por isso não pode ser comparado com TrimPrefix de string.
func deBOM(s string) string {
	b := []byte(s)
	switch {
	case len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF: // UTF-8 BOM
		b = b[3:]
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE: // UTF-16 LE BOM
		b = b[2:]
	case len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF: // UTF-16 BE BOM
		b = b[2:]
	}
	return strings.ReplaceAll(string(b), "\x00", "")
}

// cleanTokenText prepara texto colado manualmente (token.txt, campo do token)
// para extração: remove BOM/NUL, aspas envolventes, separadores invisíveis e
// o prefixo "Bearer ".
func cleanTokenText(s string) string {
	s = deBOM(s)
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	s = strings.TrimSpace(s)

	if len(s) > 7 && strings.EqualFold(s[:7], "bearer ") {
		s = strings.TrimSpace(s[7:])
	}

	// Remove aspas coladas ao token e espaços não separáveis. Mantém os pontos
	// e o traço do token intactos.
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsSpace(r), r == '\u00a0', r == '"', r == '\'':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func readTokenFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	tok, _ := tokenFromText(string(data))
	return tok
}

func readTokenJSON(path string) (string, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	tok, _ := tokenFromJSON(data)
	if tok == "" {
		return "", ""
	}
	var st storedToken
	_ = json.Unmarshal([]byte(deBOM(string(data))), &st)
	return tok, st.Account
}

func saveToken(tok, account string) {
	if tok == "" {
		return
	}
	dirs := tokenSearchDirs()
	for _, d := range dirs {
		if existingTok, existingAccount := readTokenJSON(filepath.Join(d, tokenFile)); existingTok == tok && existingAccount == account {
			return
		}
	}
	st := storedToken{Token: tok, Account: account}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		dbg("saveToken: json.Marshal: %v", err)
		return
	}
	path := filepath.Join(dirs[0], tokenFile)
	if err := os.WriteFile(path, data, 0644); err != nil && len(dirs) > 1 {
		path = filepath.Join(dirs[1], tokenFile)
		if err = os.WriteFile(path, data, 0644); err != nil {
			dbg("saveToken: erro ao guardar: %v", err)
			return
		}
	}
	dbg("token guardado em %s", path)
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
