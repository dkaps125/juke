package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type promptFunc func(string)

type model struct {
	viewport      viewport.Model
	messages      []string
	textarea      textarea.Model
	senderStyle   lipgloss.Style
	thinkingStyle lipgloss.Style
	err           error
	thinking      bool
	talking       bool
	toolCalling   bool

	onPrompt func(string)
	stopChan chan (bool)
}

func initialModel(onPrompt func(string), stopChan chan (bool)) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.SetVirtualCursor(false)
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(30), viewport.WithHeight(5))
	vp.SetContent(`Hi! What do you want to listen to today?`)
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	return model{
		textarea:      ta,
		messages:      []string{},
		viewport:      vp,
		senderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		thinkingStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#3C3C3C")),
		err:           nil,
		onPrompt:      onPrompt,
		stopChan:      stopChan,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
		m.textarea.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - m.textarea.Height())

		if len(m.messages) > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "esc":
			if m.thinking || m.talking || m.toolCalling {
				m.stopChan <- true
			}
			m.viewport.GotoBottom()
			return m, nil
		case "enter":
			m.messages = append(m.messages, m.senderStyle.Render("You: ")+m.textarea.Value())
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n\n")))
			m.viewport.GotoBottom()
			go m.onPrompt(m.textarea.Value())
			m.textarea.Reset()
			return m, nil
		default:
			// Send all other keypresses to the textarea.
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case thinkingMessage:
		if !m.thinking {
			m.talking = false
			m.thinking = true
			m.messages = append(m.messages, m.thinkingStyle.Render(string(msg)))
		} else {
			m.messages[len(m.messages)-1] = m.messages[len(m.messages)-1] + m.thinkingStyle.Render(string(msg))
		}
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n\n")))
		m.viewport.GotoBottom()
		return m, nil

	case talkingMessage:
		if !m.talking {
			m.talking = true
			m.thinking = false
			m.messages = append(m.messages, "\n"+string(msg))
		} else {
			m.messages[len(m.messages)-1] = m.messages[len(m.messages)-1] + string(msg)
		}
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n\n")))
		m.viewport.GotoBottom()
		return m, nil

	case completedMessage:
		m.talking = false
		m.thinking = false
		m.toolCalling = false
		return m, nil

	case cursor.BlinkMsg:
		// Textarea should also process cursor blinks.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() tea.View {
	viewportView := m.viewport.View()
	v := tea.NewView(viewportView + "\n" + m.textarea.View())
	c := m.textarea.Cursor()
	if c != nil {
		c.Y += lipgloss.Height(viewportView)
	}
	v.Cursor = c
	v.AltScreen = true
	return v
}
