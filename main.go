package main

import (
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	focusColor = "#2EF8BB"
	breakColor = "#FF5F87"
)

var (
	focusTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(focusColor)).MarginRight(1).SetString("Focus Mode")
	breakTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(breakColor)).MarginRight(1).SetString("Break Mode")
	pausedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(breakColor)).MarginRight(1).SetString("Continue?")
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginTop(2)
)

var baseTimerStyle = lipgloss.NewStyle().Padding(1, 2)

type mode int

const (
	Initial mode = iota
	Focusing
	Paused
	Breaking
)

type Model struct {
	quitting bool

	startTime time.Time

	mode mode

	focusTime time.Duration
	breakTime time.Duration

	progress progress.Model
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(tickInterval, tickCmd)
}

const tickInterval = time.Second / 2

type tickMsg time.Time

func tickCmd(t time.Time) tea.Msg {
	return tickMsg(t)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		cmds = append(cmds, tea.Tick(tickInterval, tickCmd))
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			switch m.mode {
			case Focusing:
				m.mode = Paused
				m.startTime = time.Now()
				m.progress.FullColor = breakColor
			case Paused:
				m.mode = Breaking
				m.startTime = time.Now()
			case Breaking:
				m.quitting = true
				return m, tea.Quit
			}
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			if m.mode == Paused {
				m.mode = Breaking
				m.startTime = time.Now()
			}
		}
	}

	// Update timer
	if m.startTime.IsZero() {
		m.startTime = time.Now()
		m.mode = Focusing
		cmds = append(cmds, tea.Tick(tickInterval, tickCmd))
	}

	switch m.mode {
	case Focusing:
		if time.Since(m.startTime) > m.focusTime {
			m.mode = Paused
			m.startTime = time.Now()
			m.progress.FullColor = breakColor
		}
	case Breaking:
		if time.Since(m.startTime) > m.breakTime {
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	elapsed := time.Since(m.startTime)

	var percent float64
	switch m.mode {
	case Focusing:
		percent = float64(elapsed) / float64(m.focusTime)
		s.WriteString(focusTitleStyle.String())
		s.WriteString(elapsed.Round(time.Second).String())
		s.WriteString("\n\n")
		s.WriteString(m.progress.ViewAs(percent))
		s.WriteString(helpStyle.Render("Press 'q' to skip"))
	case Paused:
		s.WriteString(pausedStyle.String())
		s.WriteString("\n\nFocus time is done, time to take a break.")
		s.WriteString(helpStyle.Render("press any key to continue.\n"))
	case Breaking:
		percent = float64(elapsed) / float64(m.breakTime)
		s.WriteString(breakTitleStyle.String())
		s.WriteString(elapsed.Round(time.Second).String())
		s.WriteString("\n\n")
		s.WriteString(m.progress.ViewAs(percent))
		s.WriteString(helpStyle.Render("press 'q' to quit"))
	}

	return baseTimerStyle.Render(s.String())
}

func NewModel() Model {
	progressBar := progress.New()
	progressBar.FullColor = focusColor
	progressBar.SetSpringOptions(1, 1)

	return Model{
		progress: progressBar,
	}
}

func main() {
	focusTheme := huh.ThemeCharm()
	focusTheme.Focused.Base = focusTheme.Focused.Base.Border(lipgloss.HiddenBorder())
	focusTheme.Focused.Title = focusTheme.Focused.Title.Foreground(lipgloss.Color(focusColor))
	focusTheme.Focused.SelectSelector = focusTheme.Focused.SelectSelector.Foreground(lipgloss.Color(focusColor))
	focusTheme.Focused.SelectedOption = focusTheme.Focused.SelectedOption.Foreground(lipgloss.Color("15"))
	focusTheme.Focused.Option = focusTheme.Focused.Option.Foreground(lipgloss.Color("7"))

	breakTheme := huh.ThemeCharm()
	breakTheme.Focused.Base = breakTheme.Focused.Base.Border(lipgloss.HiddenBorder())
	breakTheme.Focused.Title = breakTheme.Focused.Title.Foreground(lipgloss.Color(breakColor))
	breakTheme.Focused.SelectSelector = breakTheme.Focused.SelectSelector.Foreground(lipgloss.Color(breakColor))
	breakTheme.Focused.SelectedOption = breakTheme.Focused.SelectedOption.Foreground(lipgloss.Color("15"))
	breakTheme.Focused.Option = breakTheme.Focused.Option.Foreground(lipgloss.Color("7"))

	var (
		focusTime time.Duration
		breakTime time.Duration
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[time.Duration]().
				Title("Focus Time").
				Value(&focusTime).
				Key("focus").
				Options(
					huh.NewOption("25 minutes", 25*time.Minute),
					huh.NewOption("30 minutes", 30*time.Minute),
					huh.NewOption("45 minutes", 45*time.Minute),
					huh.NewOption("1 hour", time.Hour),
				),
		).WithTheme(focusTheme),
		huh.NewGroup(
			huh.NewSelect[time.Duration]().
				Title("Break Time").
				Value(&breakTime).
				Key("break").
				Options(
					huh.NewOption("5 minutes", 5*time.Minute),
					huh.NewOption("10 minutes", 10*time.Minute),
					huh.NewOption("15 minutes", 15*time.Minute),
					huh.NewOption("20 minutes", 20*time.Minute),
				),
		).WithTheme(breakTheme),
	).WithShowHelp(false).WithWidth(20)

	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	m := NewModel()

	m.focusTime = focusTime
	m.breakTime = breakTime

	_, err = tea.NewProgram(&m).Run()
	if err != nil {
		log.Fatal(err)
	}
}
