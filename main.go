package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	dummyRunner := flag.Bool("dummy-runner", false, "")
	cliMode := flag.Bool("cli", false, "modo terminal (sem GUI)")
	debugPtr := flag.Bool("debug", false, "modo debug com logs detalhados")
	tokenFlag := flag.String("token", "", "token do Discord (auto se omitido)")
	questID := flag.String("id", "", "ID específico de quest para processar")
	channelID := flag.String("channel", "", "ID do voice channel para heartbeats")
	stopDesktop := flag.String("stop-desktop", "", "enviar terminal heartbeat para quest PLAY_ON_DESKTOP (formato: questID,appID,exePath)")
	printToken := flag.Bool("print-token", false, "extrair e imprimir o token (útil para verificar)")
	flag.Parse()

	if *debugPtr {
		debugMode = true
	}

	if *dummyRunner {
		runDummyRunner()
		return
	}

	if *printToken {
		token, err := findDiscordToken()
		if err != nil {
			fmt.Println("✗", err)
			manualInstructions()
			os.Exit(1)
		}
		client := NewDiscordClient(token)
		if err := client.Verify(); err != nil {
			fmt.Println("✗ Token inválido:", err)
			os.Exit(1)
		}
		saveToken(token, client.user)
		fmt.Println("✓ Token válido —", client.user)
		fmt.Println("Token:", token[:30]+"...")
		return
	}

	if *stopDesktop != "" {
		parts := strings.SplitN(*stopDesktop, ",", 3)
		if len(parts) != 3 {
			fmt.Println("Formato: questID,appID,exePath")
			os.Exit(1)
		}
		stopDesktopHeartbeat(parts[0], parts[1], parts[2])
		return
	}

	if *cliMode || *questID != "" || *tokenFlag != "" {
		runCLI(*tokenFlag, *questID, *channelID)
		return
	}

	startGUI()
}

func runCLI(tokenArg, questID, channelID string) {
	fmt.Println("=== Discord Quest Emulator ===")
	fmt.Println()

	token := tokenArg
	if token == "" {
		var err error
		token, err = findDiscordToken()
		if err != nil {
			fmt.Println("✗", err)
			manualInstructions()
			os.Exit(1)
		}
		fmt.Println("✓ Token encontrado")
	} else {
		fmt.Println("✓ Token fornecido manualmente")
	}

	client := NewDiscordClient(token)
	if err := client.Verify(); err != nil {
		fmt.Println("✗", err)
		os.Exit(1)
	}
	fmt.Println("✓ Autenticado como", client.user)
	saveToken(token, client.user)

	quests, err := client.GetQuests()
	if err != nil {
		fmt.Println("✗ erro a obter quests:", err)
		os.Exit(1)
	}

	if len(quests) == 0 {
		fmt.Println("Nenhuma quest ativa encontrada.")
		return
	}

	active, completed, claimed, expired := 0, 0, 0, 0

	fmt.Printf("\nQuests encontradas (%d):\n", len(quests))
	for _, q := range quests {
		us := q.UserStatus
		name := questName(q)
		if isExpired(q) {
			fmt.Printf("  • %s [EXPIRADA]\n", name)
			expired++
			continue
		}
		if us == nil {
			fmt.Printf("  • %s [sem dados]\n", name)
			continue
		}
		if us.ClaimedAt != "" {
			fmt.Printf("  • %s [RECLAMADA]\n", name)
			claimed++
			continue
		}
		if us.CompletedAt != "" || us.Completed {
			fmt.Printf("  • %s [COMPLETA — fazer claim]\n", name)
			completed++
			continue
		}

		tasks := identifyTasks(q)
		var parts []string
		for _, t := range tasks {
			parts = append(parts, fmt.Sprintf("%s %.0f/%.0f", t.Type, t.Done, t.Target))
		}
		status := ""
		if len(parts) > 0 {
			status = " [" + join(parts, ", ") + "]"
		}
		if us.EnrolledAt == "" {
			status += " [não inscrito]"
		}
		fmt.Printf("  • %s%s\n", name, status)
		active++
	}

	if active == 0 {
		fmt.Println("\nNenhuma quest ativa e não completada.")
		if completed > 0 || claimed > 0 {
			fmt.Println("Usa --id <quest_id> para processar quests completas (claim).")
		}
		return
	}

	if channelID == "" && hasActivityQuests(quests) {
		found := client.FindVoiceChannel()
		if found != "" {
			channelID = found
		}
	}

	if questID != "" {
		var found bool
		for _, q := range quests {
			if q.ID == questID {
				runQuests(client, []Quest{q}, channelID)
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("\nQuest com ID \"%s\" não encontrada\n", questID)
			os.Exit(1)
		}
		return
	}

	fmt.Println()
	fmt.Println("A processar quests ativas...")
	runQuests(client, quests, channelID)
	fmt.Println()
	fmt.Println("Concluído! Verifica o progresso no Discord.")
}

func hasActivityQuests(quests []Quest) bool {
	for _, q := range quests {
		if q.UserStatus == nil || q.UserStatus.Completed || q.UserStatus.CompletedAt != "" {
			continue
		}
		tasks := identifyTasks(q)
		for _, t := range tasks {
			if t.Type == PlayActivity || t.Type == Achievement {
				return true
			}
		}
	}
	return false
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	for _, p := range parts[1:] {
		s += sep + p
	}
	return s
}

func runDummyRunner() {
	for {
		time.Sleep(24 * time.Hour)
	}
}

func manualInstructions() {
	fmt.Println(`Para obter o token manualmente:`)
	fmt.Println()
	fmt.Println(`Método 1 (recomendado) — Abre o Discord, Ctrl+Shift+I, vai à tab Application > Local Storage > discord.com`)
	fmt.Println(`  e copia o valor da chave "token"`)
	fmt.Println()
	fmt.Println(`Método 2 — Na tab Console, experimenta um destes:`)
	fmt.Println(`  (webpackChunkdiscord_app.push([[''],{},e=>{m=[];for(let c in e.c)m.push(e.c[c])}]),m.map(m=>m.exports).filter(x=>x?.default?.getToken?.())[0]?.default?.getToken?.())`)
	fmt.Println(`  localStorage.getItem("token")`)
	fmt.Println()
	fmt.Println(`Depois de obteres o token, guarda-o em token.json ou usa --token <token>`)
	fmt.Println()
	fmt.Println(`Dica: Se o Discord estiver aberto, o programa encontra o token automaticamente.`)
}

func stopDesktopHeartbeat(questID, appID, exePath string) {
	token, err := findDiscordToken()
	if err != nil {
		fmt.Println("✗", err)
		os.Exit(1)
	}
	client := NewDiscordClient(token)
	if err := client.Verify(); err != nil {
		fmt.Println("✗", err)
		os.Exit(1)
	}
	fmt.Println("✓ Autenticado como", client.user)
	saveToken(token, client.user)
	fmt.Println("  → A enviar terminal heartbeat...")
	_, _, err = client.SendDesktopHeartbeat(questID, appID, exePath, true)
	if err != nil {
		fmt.Println("  ✗", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Terminal heartbeat enviado!")
}
