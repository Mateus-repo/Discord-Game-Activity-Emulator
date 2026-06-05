package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var tokenFlag string
	var questID string
	var channelID string
	debugPtr := flag.Bool("debug", false, "modo debug com logs detalhados (ou -d)")
	flag.StringVar(&tokenFlag, "token", "", "token do Discord (auto se omitido)")
	flag.StringVar(&questID, "id", "", "ID específico de quest para processar")
	flag.StringVar(&channelID, "channel", "", "ID do voice channel para heartbeats")
	flag.Parse()

	if *debugPtr {
		debugMode = true
	}

	fmt.Println("=== Discord Quest Emulator ===")
	fmt.Println()

	token := tokenFlag
	if token == "" {
		var err error
		token, err = findDiscordToken()
		if err != nil {
			fmt.Println("✗", err)
			fmt.Println("Fornece o token manualmente: --token SEU_TOKEN")
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

	active := 0
	completed := 0
	claimed := 0
	expired := 0

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

	// Só procura voice channel se houver quests ACTIVITY/ACHIEVEMENT e --channel não foi dado
	if channelID == "" && hasActivityQuests(quests) {
		dbg("main: a procurar voice channel automaticamente")
		found := client.FindVoiceChannel()
		if found != "" {
			channelID = found
			fmt.Printf("✓ Voice channel encontrado: %s\n", channelID)
		} else {
			fmt.Println("! Nenhum voice channel encontrado. Usa --channel <id> ou entra num voice channel.")
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
