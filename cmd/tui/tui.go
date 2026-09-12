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

// --- styles ---

var (
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

// --- types ---

type screenKind int

const (
	screenToday screenKind = iota
	screenWeight
	screenMeals
	screenLog
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
	kind rowKind
	id   int64
	text string
	meal *db.ListMealLogsByDateRow
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

// --- model ---

type model struct {
	width  int
	height int

	screen     screenKind
	list       list.Model
	selectedRow row

	status string
	err    string
	form   formKind

	formActive bool
	formFields []textinput.Model
	formIdx    int

	out   io.Writer
	db    *db.Queries
	ctx   context.Context
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
	delegate := rowDelegate{}
	m.list = list.New(nil, delegate, m.width-22, m.height-10)
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

// --- tea.Model interface ---

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
		for _, w := range weights {
			items = append(items, row{kind: kindWeight, id: w.ID, text: fmt.Sprintf("%s  %d lbs", w.Date, w.Pounds)})
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
			if err := m.db.UpdateWeight(m.ctx, db.UpdateWeightParams{Pounds: int64(lbs), Date: d}); err == nil {
				m.status = fmt.Sprintf("updated weight to %d lbs", lbs)
			} else if errors.Is(err, sql.ErrNoRows) {
				if _, err := m.db.CreateWeight(m.ctx, db.CreateWeightParams{Pounds: int64(lbs), Date: d}); err != nil {
					return errMsg{err}
				}
				m.status = fmt.Sprintf("logged weight %d lbs", lbs)
			} else {
				return errMsg{err}
			}
			m.cancelForm()
			return m.refreshList()
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
			return m.refreshList()
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
			return m.refreshList()
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
		return m.refreshList()
	}
}

func (m *model) View() string {
	navWidth := 18
	headerHeight := 1
	helpHeight := 1
	contentHeight := m.height - headerHeight - helpHeight - 2

	// Header
	headerText := " diet-tracker tui "
	if m.screen != screenToday {
		headerText += "  " + navItemSelected.Render(fmt.Sprintf("[1:%s] [2:%s] [3:%s] [4:%s]",
			screenToday, screenWeight, screenMeals, screenLog))
	}
	header := lipgloss.NewStyle().
		Width(m.width).
		Height(headerHeight).
		Render(headerStyle.Render(headerText))

	// Nav panel
	var navItems strings.Builder
	navItems.WriteString(navActive.Render(" Navigation "))
	navItems.WriteString("\n")
	screens := []screenKind{screenToday, screenWeight, screenMeals, screenLog}
	for _, s := range screens {
		icon := s.Icon()
		label := s.String()
		selected := s == m.screen
		item := fmt.Sprintf("  %s  %-14s", icon, label)
		if selected {
			navItems.WriteString(navItemSelected.Render(item) + " ←")
		} else {
			navItems.WriteString(navItemUnselected.Render(item))
		}
		navItems.WriteString("\n")
	}
	navItems.WriteString("\n")
	navItems.WriteString(helpStyle.Render("  [n] new"))
	navItems.WriteString("\n")
	navItems.WriteString(helpStyle.Render("  [x] del"))
	navItems.WriteString("\n")
	navItems.WriteString(helpStyle.Render("  [q] quit"))

	navPanel := lipgloss.NewStyle().
		Width(navWidth).
		Height(contentHeight).
		Render(navItems.String())

	// Content panel
	contentWidth := m.width - navWidth - 1

	var content strings.Builder
	if m.formActive {
		// Modal form centered in content area
		formWidth := 40
		formHeight := 6
		modalX := (contentWidth - formWidth) / 2
		modalY := (contentHeight - formHeight) / 2

		for y := 0; y < contentHeight; y++ {
			if y < modalY {
				content.WriteString(strings.Repeat(" ", contentWidth))
			} else if y == modalY {
				content.WriteString(strings.Repeat(" ", modalX))
				content.WriteString(modalStyle.BorderTop(true).Render(strings.Repeat(" ", formWidth)))
			} else if y == modalY+formHeight-1 {
				content.WriteString(strings.Repeat(" ", modalX))
				content.WriteString(modalStyle.BorderBottom(true).Render(strings.Repeat(" ", formWidth)))
			} else if y > modalY && y < modalY+formHeight-1 {
				content.WriteString(strings.Repeat(" ", modalX))
				row := y - modalY
				switch row {
				case 0:
					title := ""
					switch m.form {
					case formWeight:
						title = " Log Weight "
					case formMeal:
						title = " Add Meal   "
					case formLog:
						title = " Log Exercise"
					}
					content.WriteString(modalStyle.Render(title))
				case 1:
					content.WriteString(modalStyle.Render(strings.Repeat("-", formWidth-2)))
				case 2:
					for i, f := range m.formFields {
						prompt := f.Prompt
						if i == m.formIdx {
							prompt = "> " + prompt
						}
						val := f.View()
						if i == m.formIdx {
							val = inputStyle.Render(val)
						}
						content.WriteString(modalStyle.Render(fmt.Sprintf("  %-10s%s", prompt, val)))
					}
				case 3:
					content.WriteString(modalStyle.Render("  tab: next · esc: cancel"))
				default:
					content.WriteString(modalStyle.Render(strings.Repeat(" ", formWidth-2)))
				}
			} else {
				content.WriteString(strings.Repeat(" ", contentWidth))
			}
			content.WriteString("\n")
		}
	} else {
		// Normal list view
		listView := m.list.View()
		if m.status != "" {
			listView += "\n" + statusStyle.Render(" ✓ " + m.status)
		}
		if m.err != "" {
			listView += "\n" + errStyle.Render(" ⚠ " + m.err)
		}
		content.WriteString(contentBorder.Render(listView))
	}

	contentPanel := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Render(content.String())

	// Help bar
	helpBar := lipgloss.NewStyle().
		Width(m.width).
		Height(helpHeight).
		Render(helpStyle.Render(" n: new  x: delete  q: quit"))

	// Assemble layout
	layout := lipgloss.JoinVertical(lipgloss.Top,
		header,
		lipgloss.JoinHorizontal(lipgloss.Top, navPanel, contentPanel),
		helpBar,
	)

	return layout
}


func (m *model) Run() {
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil && m.out != nil {
		panic(err)
	}
}

// --- helpers ---

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
type msg struct{}

type rowDelegate struct{ list.DefaultDelegate }

func (rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	r, ok := item.(row)
	if !ok {
		return
	}
	fmt.Fprint(w, r.text)
}
