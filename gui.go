package main

import (
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type questItem struct {
	quest Quest
	tasks []TaskInfo

	check   *widget.Check
	nameLbl *widget.Label
	bar     *widget.ProgressBar
}

type gui struct {
	win            fyne.Window
	a              fyne.App
	client         *DiscordClient

	statusLbl      *widget.Label
	voiceLbl       *widget.Label
	voiceChannelID string
	questBox       *fyne.Container
	logW           *widget.Entry
	processBtn     *widget.Button
	refreshBtn     *widget.Button

	items   []*questItem
	quests  []Quest
	mu      sync.Mutex
	running bool
}

func startGUI() {
	g := &gui{
		items: make([]*questItem, 0),
	}
	g.a = app.New()
	g.win = g.a.NewWindow("Discord Quest Emulator")
	g.win.Resize(fyne.NewSize(720, 620))

	setProgressChannel(make(chan ProgressUpdate, 64))
	logFn = g.log

	g.buildUI()
	go g.progressListener()
	g.win.ShowAndRun()
}

func (g *gui) buildUI() {
	title := canvas.NewText("Discord Quest Emulator", color.RGBA{0x58, 0x65, 0xF2, 0xFF})
	title.TextSize = 20

	g.statusLbl = widget.NewLabel("A iniciar...")

	g.voiceLbl = widget.NewLabel("")
	g.voiceLbl.Hide()

	authVBox := container.NewVBox(
		widget.NewCard("", "Autenticação",
			container.NewBorder(nil, nil, widget.NewLabel("Token:"), nil,
				widget.NewLabel("(detetado automaticamente)"),
			),
		),
		g.statusLbl,
		g.voiceLbl,
	)

	g.questBox = container.NewVBox()
	questScroll := container.NewScroll(g.questBox)
	questScroll.SetMinSize(fyne.NewSize(0, 200))

	g.processBtn = widget.NewButtonWithIcon("Processar", theme.MediaPlayIcon(), g.onProcess)
	g.refreshBtn = widget.NewButtonWithIcon("Atualizar", theme.ViewRefreshIcon(), g.onRefresh)
	controls := container.NewHBox(layout.NewSpacer(), g.processBtn, g.refreshBtn)

	questCard := widget.NewCard("", "Quests",
		container.NewBorder(nil, controls, nil, nil, questScroll),
	)

	g.logW = widget.NewMultiLineEntry()
	g.logW.SetMinRowsVisible(8)
	g.logW.Wrapping = fyne.TextWrapWord
	logCard := widget.NewCard("", "Log", g.logW)

	split := container.NewVSplit(questCard, logCard)

	g.win.SetContent(container.NewBorder(
		container.NewVBox(title, authVBox),
		nil, nil, nil,
		split,
	))

	g.onRefresh()
}

func (g *gui) log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fyne.Do(func() {
		g.logW.Append(msg + "\n")
	})
}

func (g *gui) onRefresh() {
	if g.running {
		return
	}
	g.statusLbl.SetText("A procurar token...")

	client, err := g.auth()
	if err != nil {
		g.statusLbl.SetText("✗ " + err.Error())
		return
	}
	g.client = client

	quests, err := client.GetQuests()
	if err != nil {
		g.statusLbl.SetText("✗ Erro: " + err.Error())
		return
	}

	g.mu.Lock()
	g.quests = quests
	g.rebuildQuestList()
	g.mu.Unlock()

	if len(quests) == 0 {
		g.statusLbl.SetText("Autenticado, sem quests ativas.")
		return
	}

	active, completed, claimed := 0, 0, 0
	for _, q := range quests {
		if q.UserStatus == nil {
			continue
		}
		if isExpired(q) {
			continue
		}
		if q.UserStatus.ClaimedAt != "" {
			claimed++
			continue
		}
		if q.UserStatus.Completed || q.UserStatus.CompletedAt != "" {
			completed++
			continue
		}
		active++
	}

	parts := []string{}
	if active > 0 {
		parts = append(parts, fmt.Sprintf("%d ativas", active))
	}
	if completed > 0 {
		parts = append(parts, fmt.Sprintf("%d completas", completed))
	}
	if claimed > 0 {
		parts = append(parts, fmt.Sprintf("%d reclamadas", claimed))
	}
	g.statusLbl.SetText(fmt.Sprintf("✓ Autenticado como %s — %s", client.user, strings.Join(parts, ", ")))

	g.findVoiceChannel()
}

func (g *gui) auth() (*DiscordClient, error) {
	token, err := findDiscordToken()
	if err != nil {
		return nil, fmt.Errorf(`token não encontrado.

1. Abre o Discord, Ctrl+Shift+I, tab Console
2. Cola um destes comandos:

   (webpackChunkdiscord_app.push([[''],{},e=>{m=[];for(let c in e.c)m.push(e.c[c])}]),m.map(m=>m.exports).filter(x=>x?.default?.getToken?.())[0]?.default?.getToken?.())

   Se der undefined, tenta:
   (function(){const w=webpackChunkdiscord_app.push([[],{},r=>r]);delete w.default;const m=Object.values(w.c).find(m=>m?.exports?.default?.getToken);return m?.exports?.default?.getToken()})()

3. Guarda o resultado em token.json e reabre o programa`)
	}
	client := NewDiscordClient(token)
	if err := client.Verify(); err != nil {
		return nil, err
	}
	return client, nil
}

func (g *gui) rebuildQuestList() {
	g.questBox.Objects = nil
	g.questBox.Refresh()
	g.items = nil

	for _, q := range g.quests {
		if q.UserStatus == nil {
			continue
		}
		if isExpired(q) {
			continue
		}
		if q.UserStatus.ClaimedAt != "" {
			continue
		}

		tasks := identifyTasks(q)
		item := &questItem{
			quest: q,
			tasks: tasks,
		}
		g.items = append(g.items, item)
		g.questBox.Add(g.makeQuestCard(item))
	}
	g.questBox.Refresh()
}

func (g *gui) makeQuestCard(item *questItem) *fyne.Container {
	q := item.quest
	us := q.UserStatus
	name := questName(q)

	var metaParts []string
	for _, t := range item.tasks {
		metaParts = append(metaParts, fmt.Sprintf("%s %.0f/%.0f", t.Type, t.Done, t.Target))
	}

	completed := us.Completed || us.CompletedAt != ""
	claimed := us.ClaimedAt != ""

	item.check = widget.NewCheck("", nil)
	if completed || claimed {
		item.check.SetChecked(true)
		item.check.Disable()
	} else {
		item.check.SetChecked(true)
	}

	statusStr := ""
	switch {
	case claimed:
		statusStr = "Reclamada"
	case completed:
		statusStr = "Completa"
	default:
		statusStr = "Ativa"
	}

	item.nameLbl = widget.NewLabel(name)
	item.bar = widget.NewProgressBar()

	if completed || claimed {
		item.bar.SetValue(1.0)
	} else {
		for _, t := range item.tasks {
			if t.Target > 0 {
				item.bar.SetValue(t.Done / t.Target)
				break
			}
		}
	}

	metaStr := strings.Join(metaParts, ", ")
	if metaStr == "" {
		metaStr = statusStr
	} else {
		metaStr += " — " + statusStr
	}
	metaLbl := widget.NewLabel(metaStr)
	metaLbl.TextStyle = fyne.TextStyle{Monospace: true}

	return container.NewVBox(widget.NewCard("", "", container.NewVBox(
		container.NewHBox(item.check, item.nameLbl),
		item.bar,
		metaLbl,
	)))
}

func (g *gui) findVoiceChannel() {
	if g.client == nil {
		return
	}
	ch := g.client.FindVoiceChannel()
	g.voiceChannelID = ch
	if ch != "" {
		g.voiceLbl.SetText("✓ Voice channel: " + ch)
		g.voiceLbl.Show()
	} else {
		g.voiceLbl.SetText("! Nenhum voice channel. ACTIVITY pode falhar.")
		g.voiceLbl.Show()
	}
}

func (g *gui) onProcess() {
	if g.running || g.client == nil || len(g.items) == 0 {
		return
	}

	var selected []Quest
	g.mu.Lock()
	for _, item := range g.items {
		if item.check.Checked {
			selected = append(selected, item.quest)
		}
	}
	g.mu.Unlock()

	if len(selected) == 0 {
		return
	}

	g.running = true
	g.processBtn.SetText("A processar...")
	g.processBtn.Disable()
	g.refreshBtn.Disable()

	go func() {
		runQuests(g.client, selected, g.voiceChannelID)

		time.Sleep(1 * time.Second)
		fyne.Do(func() {
			g.running = false
			g.processBtn.SetText("Processar")
			g.processBtn.Enable()
			g.refreshBtn.Enable()
			g.onRefresh()
		})
	}()
}

func (g *gui) progressListener() {
	ch := progressCh
	if ch == nil {
		return
	}
	for pu := range ch {
		val := 0.0
		if pu.Target > 0 {
			val = pu.Current / pu.Target
			if val > 1.0 {
				val = 1.0
			}
		}
		var bar *widget.ProgressBar
		g.mu.Lock()
		for _, item := range g.items {
			if item.quest.ID == pu.QuestID {
				bar = item.bar
				break
			}
		}
		g.mu.Unlock()
		if bar != nil {
			fyne.Do(func() {
				bar.SetValue(val)
			})
		}
	}
}
