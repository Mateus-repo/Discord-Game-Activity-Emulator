package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	cliMode := flag.Bool("cli", false, "modo terminal (sem GUI)")
	debugPtr := flag.Bool("debug", false, "modo debug com logs detalhados")
	tokenFlag := flag.String("token", "", "token do Discord (auto se omitido)")
	questID := flag.String("id", "", "ID específico de quest para processar")
	channelID := flag.String("channel", "", "ID do voice channel para heartbeats")
	flag.Parse()

	if *debugPtr {
		debugMode = true
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
			fmt.Println("\nObtém o token manualmente:")
			fmt.Println("1. Abre Discord (Ctrl+Shift+I)")
			fmt.Println("2. Vai à tab Console")
			fmt.Println(`3. Cola: (webpackChunkdiscord_app.push([[''],{},e=>{m=[];for(let c in e.c)m.push(e.c[c])}]),m.map(m=>m.exports).filter(x=>x?.default?.getToken?.())[0]?.default?.getToken?.())`)
			fmt.Println("4. Guarda o resultado em token.txt ou usa --token <token>")
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
