package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dukedelaet/diet-tracker/internal/db"
)

var (
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Background(lipgloss.Color("237")).
		Padding(0, 1)

	navItemSelected = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Padding(0, 1)

	navItemUnselected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	navActive = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("63")).
		Padding(0, 1)

	contentBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("236")).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("51"))

	errStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	modalStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Border(lipgloss.ThickBorder()).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(1, 2)

	inputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))
)

type screenKind int

const (
	screenToday screenKind = iota
	screenWeight
	screenMeals
	screenLog
	numScreens
)

func (s screenKind) String() string {
	switch s {
	case screenToday:
		return "today"
	case screenWeight:
		return "weight"
	case screenMeals:
		return "meals"
	case screenLog:
		return "log"
	}
	return ""
}

func (s screenKind) Icon() string {
	switch s {
	case screenToday:
		return "◉"
	case screenWeight:
		return "⚖"
	case screenMeals:
		return "🍽"
	case screenLog:
		return "▶"
	}
	return "·"
}

type rowKind int

const (
	kindWeight rowKind = iota
	kindMealLog
	kindExercise
	kindTotal
)

type row struct {
	kind   rowKind
	id     int64
	text   string
	pounds int64
	meal   *db.ListMealLogsByDateRow
}

func (r row) title() string        { return r.text }
func (r row) description() string { return "" }
func (r row) FilterValue() string { return "" }

type formKind int

const (
	formNone formKind = iota
	formWeight
	formMeal
	formLog
)

type model struct {
	width, height int
	screen        screenKind
	list          list.Model
	selectedRow   row
	status        string
	err           string
	form          formKind
	formActive    bool
	formFields    []textinput.Model
	formIdx       int
	out           io.Writer
	db            *db.Queries
	ctx           context.Context
	weights       []row
}

type NewModelOpts struct {
	DB     *sql.DB
	Out    io.Writer
	Ctx    context.Context
	Width  int
	Height int
}

func NewModel(opts NewModelOpts) *model {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}
	if opts.Width == 0 {
		opts.Width = 80
	}
	if opts.Height == 0 {
		opts.Height = 24
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	m := &model{
		width:  opts.Width,
		height: opts.Height,
		out:    opts.Out,
		db:     db.New(opts.DB),
		ctx:    opts.Ctx,
	}
	m.setupList()
	return m
}

func (m *model) setupList() {
	m.list = list.New(nil, rowDelegate{}, m.width-22, m.height-10)
	m.list.SetShowHelp(false)
	m.list.SetShowStatusBar(false)
	m.list.Title = ""
}

func (m *model) newFormFields(kind formKind) []textinput.Model {
	switch kind {
	case formWeight:
		lbs := textinput.New()
		lbs.Prompt = "lbs: "
		lbs.CharLimit = 5
		return []textinput.Model{lbs}
	case formMeal:
		name := textinput.New()
		name.Prompt = "name: "
		p := textinput.New()
		p.Prompt = "p: "
		p.CharLimit = 4
		c := textinput.New()
		c.Prompt = "c: "
		c.CharLimit = 4
		f := textinput.New()
		f.Prompt = "f: "
		f.CharLimit = 4
		return []textinput.Model{name, p, c, f}
	case formLog:
		t := textinput.New()
		t.Prompt = "type: "
		t.SetValue("cardio")
		d := textinput.New()
		d.Prompt = "min: "
		d.CharLimit = 4
		d.SetValue("30")
		return []textinput.Model{t, d}
	}
	return nil
}

func (m *model) startForm(kind formKind) {
	m.form = kind
	m.formActive = true
	m.formFields = m.newFormFields(kind)
	m.formIdx = 0
	m.err = ""
	m.formFields[0].Focus()
}

func (m *model) cancelForm() {
	m.formActive = false
	m.form = formNone
	m.formFields = nil
	m.formIdx = 0
	m.err = ""
}

func (m *model) switchScreen(s screenKind) {
	m.screen = s
	m.cancelForm()
	m.status = ""
}

func (m *model) Init() tea.Cmd {
	return m.refreshList()
}

func (m *model) refreshList() tea.Cmd {
	switch m.screen {
	case screenToday:
		return m.loadToday()
	case screenWeight:
		return m.loadRecentWeights()
	case screenMeals:
		return m.loadMeals()
	case screenLog:
		return m.loadLog()
	}
	return nil
}

func (m *model) loadToday() tea.Cmd {
	q, ctx, d := m.db, m.ctx, todayStr()
	return func() tea.Msg {
		var items []list.Item
		w, err := q.GetWeightByDate(ctx, d)
		switch {
		case err == nil:
			items = append(items, row{kind: kindWeight, id: w.ID, text: fmt.Sprintf("Weight  %d lbs", w.Pounds)})
		case !errors.Is(err, sql.ErrNoRows):
			return errMsg{err}
		}
		logs, err := q.ListMealLogsByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		for _, l := range logs {
			l := l
			items = append(items, row{kind: kindMealLog, id: l.ID, meal: &l,
				text: fmt.Sprintf("%-24s  %dP %dC %dF  %dcals", l.Name, l.Protein, l.Carbs, l.Fat, l.Calories)})
		}
		exs, err := q.ListExercisesByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		for _, e := range exs {
			items = append(items, row{kind: kindExercise, id: e.ID,
				text: fmt.Sprintf("%s  %d min", e.ExerciseType, e.Duration)})
		}
		tot, err := q.GetDailyTotals(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		items = append(items, row{kind: kindTotal,
			text: fmt.Sprintf("Totals  %dP %dC %dF  %dcals", float64AsInt(tot.TotalProtein), float64AsInt(tot.TotalCarbs), float64AsInt(tot.TotalFat), float64AsInt(tot.TotalCalories))})
		m.list.Title = "Today " + d
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) loadRecentWeights() tea.Cmd {
	q, ctx := m.db, m.ctx
	return func() tea.Msg {
		weights, err := q.ListWeights(ctx, 30)
		if err != nil {
			return errMsg{err}
		}
		var items []list.Item
		m.weights = nil
		for _, w := range weights {
			r := row{kind: kindWeight, id: w.ID, text: fmt.Sprintf("%s  %d lbs", w.Date, w.Pounds), pounds: w.Pounds}
			items = append(items, r)
			m.weights = append(m.weights, r)
		}
		m.list.Title = "Recent weights"
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) loadMeals() tea.Cmd {
	q, ctx := m.db, m.ctx
	d := todayStr()
	return func() tea.Msg {
		logs, err := q.ListMealLogsByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		var items []list.Item
		for _, l := range logs {
			l := l
			items = append(items, row{kind: kindMealLog, id: l.ID, meal: &l,
				text: fmt.Sprintf("%-24s  %dP %dC %dF  %dcals", l.Name, l.Protein, l.Carbs, l.Fat, l.Calories)})
		}
		m.list.Title = "Meals logged today"
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) loadLog() tea.Cmd {
	q, ctx, d := m.db, m.ctx, todayStr()
	return func() tea.Msg {
		exs, err := q.ListExercisesByDate(ctx, d)
		if err != nil {
			return errMsg{err}
		}
		var items []list.Item
		for _, e := range exs {
			items = append(items, row{kind: kindExercise, id: e.ID,
				text: fmt.Sprintf("%s  %d min", e.ExerciseType, e.Duration)})
		}
		m.list.Title = "Exercises today"
		m.list.SetItems(items)
		return msg{}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.formActive {
			return m.handleFormKey(msg)
		}
		return m.handleNavKey(msg)
	}
	l, cmd := m.list.Update(msg)
	m.list = l
	return m, cmd
}

func (m *model) handleNavKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case k.Type == tea.KeyCtrlC:
		return m, tea.Quit
	case k.String() == "q":
		return m, tea.Quit
	case k.Type == tea.KeyUp || k.String() == "k":
		m.screen = (m.screen - 1 + numScreens) % numScreens
		return m, m.refreshList()
	case k.Type == tea.KeyDown || k.String() == "j":
		m.screen = (m.screen + 1) % numScreens
		return m, m.refreshList()
	case k.String() == "1":
		m.switchScreen(screenToday)
		return m, m.refreshList()
	case k.String() == "2":
		m.switchScreen(screenWeight)
		return m, m.refreshList()
	case k.String() == "3":
		m.switchScreen(screenMeals)
		return m, m.refreshList()
	case k.String() == "4":
		m.switchScreen(screenLog)
		return m, m.refreshList()
	case k.String() == "n":
		switch m.screen {
		case screenWeight:
			m.startForm(formWeight)
		case screenMeals:
			m.startForm(formMeal)
		case screenLog:
			m.startForm(formLog)
		}
		return m, nil
	case k.String() == "x":
		if cur, ok := m.list.SelectedItem().(row); ok {
			return m.deleteRow(cur)
		}
	}
	l, cmd := m.list.Update(k)
	m.list = l
	return m, cmd
}

func (m *model) handleFormKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case k.Type == tea.KeyCtrlC:
		m.cancelForm()
		return m, m.refreshList()
	case k.Type == tea.KeyEsc:
		m.cancelForm()
		return m, m.refreshList()
	case k.Type == tea.KeyEnter && m.formIdx == len(m.formFields)-1:
		return m.submitForm()
	case k.Type == tea.KeyTab:
		m.formIdx = (m.formIdx + 1) % len(m.formFields)
		return m, m.formFields[m.formIdx].Focus()
	case k.Type == tea.KeyShiftTab:
		m.formIdx = (m.formIdx - 1 + len(m.formFields)) % len(m.formFields)
		return m, m.formFields[m.formIdx].Focus()
	}
	ti, cmd := m.formFields[m.formIdx].Update(k)
	m.formFields[m.formIdx] = ti
	return m, cmd
}

func intOr(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	return strconv.Atoi(s)
}

func (m *model) submitForm() (tea.Model, tea.Cmd) {
	f := m.formFields
	switch m.form {
	case formWeight:
		lbs, err := intOr(f[0].Value(), 0)
		if err != nil || lbs <= 0 {
			m.err = "invalid pounds"
			return m, nil
		}
		return m, func() tea.Msg {
			d := todayStr()
			_, getErr := m.db.GetWeightByDate(m.ctx, d)
			if errors.Is(getErr, sql.ErrNoRows) {
				if _, err := m.db.CreateWeight(m.ctx, db.CreateWeightParams{Pounds: int64(lbs), Date: d}); err != nil {
					return errMsg{err}
				}
				m.status = fmt.Sprintf("logged weight %d lbs", lbs)
			} else if getErr != nil {
				return errMsg{getErr}
			} else {
				if err := m.db.UpdateWeight(m.ctx, db.UpdateWeightParams{Pounds: int64(lbs), Date: d}); err != nil {
					return errMsg{err}
				}
				m.status = fmt.Sprintf("updated weight to %d lbs", lbs)
			}
			m.cancelForm()
			return reloadMsg{}
		}
	case formMeal:
		name := f[0].Value()
		p, e1 := intOr(f[1].Value(), 0)
		c, e2 := intOr(f[2].Value(), 0)
		fat, e3 := intOr(f[3].Value(), 0)
		if name == "" || e1 != nil || e2 != nil || e3 != nil {
			m.err = "fill name and all three macros"
			return m, nil
		}
		if p+c+fat == 0 {
			m.err = "macros cannot all be zero"
			return m, nil
		}
		cals := c*4 + p*4 + fat*9
		return m, func() tea.Msg {
			if _, err := m.db.GetMealByName(m.ctx, name); err == nil {
				return errMsg{errors.New("meal already exists")}
			} else if !errors.Is(err, sql.ErrNoRows) {
				return errMsg{err}
			}
			if _, err := m.db.CreateMeal(m.ctx, db.CreateMealParams{Name: name, Protein: int64(p), Carbs: int64(c), Fat: int64(fat), Calories: int64(cals)}); err != nil {
				return errMsg{err}
			}
			m.status = fmt.Sprintf("created meal %s (%d cals)", name, cals)
			m.cancelForm()
			return reloadMsg{}
		}
	case formLog:
		t := strings.ToLower(strings.TrimSpace(f[0].Value()))
		dur, err := intOr(f[1].Value(), 0)
		if err != nil || (t != "cardio" && t != "strength") || dur <= 0 {
			m.err = "type must be cardio/strength, minutes > 0"
			return m, nil
		}
		return m, func() tea.Msg {
			if _, err := m.db.CreateExercise(m.ctx, db.CreateExerciseParams{Date: todayStr(), ExerciseType: t, Duration: int64(dur)}); err != nil {
				return errMsg{err}
			}
			m.status = fmt.Sprintf("logged %d min %s", dur, t)
			f[1].SetValue("30")
			f[0].SetValue("cardio")
			m.cancelForm()
			return reloadMsg{}
		}
	}
	return m, nil
}

func (m *model) deleteRow(r row) (tea.Model, tea.Cmd) {
	id := r.id
	deleteFn := func() error {
		switch r.kind {
		case kindWeight:
			return m.db.DeleteWeight(m.ctx, id)
		case kindMealLog:
			return m.db.DeleteMealLog(m.ctx, id)
		case kindExercise:
			return m.db.DeleteExercise(m.ctx, id)
		default:
			return nil
		}
	}
	return m, func() tea.Msg {
		if err := deleteFn(); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return errMsg{err}
		}
		m.status = "deleted entry"
		return reloadMsg{}
	}
}

func (m *model) View() string {
	navWidth := 16
	headerHeight := 1
	helpHeight := 1
	bodyHeight := m.height - headerHeight - helpHeight
	contentWidth := m.width - navWidth

	title := headerStyle.Render(" diet-tracker tui ")
	header := lipgloss.NewStyle().Width(m.width).Height(headerHeight).Render(title)

	var navLines []string
	navLines = append(navLines, navActive.Render(navLabel(" DIET ", navWidth)))
	for _, s := range []screenKind{screenToday, screenWeight, screenMeals, screenLog} {
		label := fmt.Sprintf(" %s %s", s.Icon(), s.String())
		if s == m.screen {
			navLines = append(navLines, navActive.Render(navLabel(label, navWidth)))
		} else {
			navLines = append(navLines, navItemUnselected.Render(navLabel(label, navWidth)))
		}
	}
	for i := len(navLines); i < bodyHeight; i++ {
		navLines = append(navLines, strings.Repeat(" ", navWidth))
	}
	navPanel := strings.Join(navLines[:bodyHeight], "\n")

	var contentView string
	if m.formActive {
		contentView = lipgloss.Place(contentWidth, bodyHeight,
			lipgloss.Center, lipgloss.Center, m.modalLines())
	} else if m.screen == screenWeight && len(m.weights) > 0 {
		chart := m.renderWeightChart(contentWidth-4, bodyHeight-4)
		contentView = chart
	} else {
		listWidth := contentWidth - 4
		listHeight := bodyHeight - 2
		if listHeight < 1 {
			listHeight = 1
		}
		m.list.SetSize(listWidth, listHeight)
		body := m.list.View()
		if m.err != "" {
			body += "\n" + errStyle.Render(" "+m.err)
		}
		if m.status != "" {
			body += "\n" + statusStyle.Render(" "+m.status)
		}
		contentView = contentBorder.Copy().Width(contentWidth-2).Height(listHeight+2).Render(body)
	}

	helpText := " j/k or ↑↓ nav   n new   x delete   q quit"
	if m.formActive {
		helpText = " tab next   enter submit   esc cancel"
	}
	helpBar := lipgloss.NewStyle().Width(m.width).Height(helpHeight).Render(helpStyle.Render(helpText))

	return lipgloss.JoinVertical(lipgloss.Top, header,
		lipgloss.JoinHorizontal(lipgloss.Top, navPanel, contentView),
		helpBar)
}

func (m *model) renderWeightChart(width, height int) string {
	if len(m.weights) < 2 {
		return "No enough data for chart"
	}
	
	// Find min/max
	minVal, maxVal := float64(m.weights[0].pounds), float64(m.weights[0].pounds)
	for _, w := range m.weights {
		if float64(w.pounds) < minVal {
			minVal = float64(w.pounds)
		}
		if float64(w.pounds) > maxVal {
			maxVal = float64(w.pounds)
		}
	}
	if maxVal == minVal {
		maxVal++
	}
	
	// Chart dimensions (leave room for axes)
	charWidth := width - 4
	charHeight := height - 2
	
	var lines []string
	
	// Top border
	lines = append(lines, "┌"+strings.Repeat("─", charWidth)+"┐")
	
	// Draw chart rows
	for row := charHeight - 1; row >= 0; row-- {
		line := "│"
		// Calculate y value for this row
		yVal := minVal + float64(row)/float64(charHeight-1)*float64(maxVal-minVal)
		line += fmt.Sprintf("%4.0f ", yVal)
		
		// Plot points
		for col := 0; col < charWidth-2; col++ {
			idx := col * (len(m.weights) - 1) / (charWidth - 3)
			if idx >= len(m.weights) {
				idx = len(m.weights) - 1
			}
			wVal := float64(m.weights[idx].pounds)
			rowAtVal := int((wVal - minVal) / (maxVal - minVal) * float64(charHeight-1))
			if rowAtVal == row {
				line += "█"
			} else {
				line += " "
			}
		}
		line += "│"
		lines = append(lines, line)
	}
	
	// Bottom border
	lines = append(lines, "└"+strings.Repeat("─", charWidth)+"┘")
	
	return strings.Join(lines, "\n")
}

func navLabel(s string, w int) string {
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func (m *model) modalLines() string {
	var fields strings.Builder
	for i, f := range m.formFields {
		prompt := f.Prompt
		val := f.View()
		if i == m.formIdx {
			fields.WriteString(inputStyle.Render("> "+prompt+val) + "\n")
		} else {
			fields.WriteString(dimStyle.Render("  "+prompt+val) + "\n")
		}
	}
	title := ""
	switch m.form {
	case formWeight:
		title = "Log weight (lbs)"
	case formMeal:
		title = "New meal"
	case formLog:
		title = "Log exercise"
	}
	inner := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render(title),
		strings.TrimRight(fields.String(), "\n"),
		dimStyle.Render("tab next · enter save · esc cancel"),
	)
	return modalStyle.Render(inner)
}

func (m *model) Run() {
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil && m.out != nil {
		panic(err)
	}
}

func todayStr() string {
	return time.Now().In(time.Local).Format(time.DateOnly)
}

func float64AsInt(v interface{}) int {
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return int(f)
}

type errMsg struct{ err error }
type reloadMsg struct{}
type msg struct{}

type rowDelegate struct{ list.DefaultDelegate }

func (rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	r, ok := item.(row)
	if !ok {
		return
	}
	fmt.Fprint(w, r.text)
}
