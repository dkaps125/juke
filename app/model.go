package app

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dkaps125/juke/harness"
	"github.com/dkaps125/juke/utils"
)

var items = []list.Item{item("/compact"), item("/play"), item("/pause")}

type model struct {
	viewport         viewport.Model
	messages         []string
	textarea         textarea.Model
	slashCommandList list.Model
	senderStyle      lipgloss.Style
	thinkingStyle    lipgloss.Style
	err              error
	thinking         bool
	talking          bool
	toolCalling      bool

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

	scl := list.New(items, itemDelegate{lipgloss.NewStyle().Foreground(lipgloss.Color("#3C3C3C"))}, 20, 0)
	scl.SetShowTitle(false)
	scl.SetShowStatusBar(false)
	scl.SetFilteringEnabled(false)
	scl.SetShowHelp(false)
	scl.SetShowPagination(false)

	return model{
		textarea:         ta,
		messages:         []string{},
		viewport:         vp,
		slashCommandList: scl,
		senderStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		thinkingStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#3C3C3C")),
		err:              nil,
		onPrompt:         onPrompt,
		stopChan:         stopChan,
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
		m.viewport.SetHeight(msg.Height - m.textarea.Height() - m.slashCommandList.Height())

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
			return m.hideCommandList()
		default:
			var textAreaCmd tea.Cmd
			m.textarea, textAreaCmd = m.textarea.Update(msg)

			textAreaValue := m.textarea.Value()

			// Expand the command list
			if len(textAreaValue) == 1 && msg.String() == "/" {
				var listCmd tea.Cmd
				m.slashCommandList.SetHeight(4)
				m.slashCommandList, listCmd = m.slashCommandList.Update(msg)
				m.viewport.SetHeight(m.viewport.Height() - m.slashCommandList.Height())
				return m, tea.Batch(textAreaCmd, listCmd)
			}

			// Handle keypresses with the command list open
			if m.slashCommandList.Height() > 0 {
				// If at any point the first character of the text area is no longer a slash, collapse the command list
				if len(textAreaValue) == 0 || string(textAreaValue[0]) != "/" {
					return m.hideCommandList()
				}

				if msg.Code == tea.KeyUp || msg.Code == tea.KeyDown {
					var cmd tea.Cmd
					m.slashCommandList, cmd = m.slashCommandList.Update(msg)
					m.textarea.SetValue(string(m.slashCommandList.SelectedItem().(item)))
					return m, cmd
				}

				m.slashCommandList.SetItems(utils.Filter(items, func(item list.Item) bool {
					return strings.HasPrefix(fmt.Sprintf("%s", item), textAreaValue)
				}))

				var cmd tea.Cmd
				m.slashCommandList, cmd = m.slashCommandList.Update(msg)
				return m, cmd
			}

			return m, textAreaCmd
		}

	case harness.ThinkingMessage:
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

	case harness.TalkingMessage:
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

	case harness.CompletedMessage:
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

	var cmd tea.Cmd
	m.slashCommandList, cmd = m.slashCommandList.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	viewportView := m.viewport.View()
	v := tea.NewView(viewportView + "\n" + m.textarea.View() + "\n" + m.slashCommandList.View())
	c := m.textarea.Cursor()
	if c != nil {
		c.Y += lipgloss.Height(viewportView)
	}
	v.Cursor = c
	v.AltScreen = true
	return v
}

func (m model) hideCommandList() (tea.Model, tea.Cmd) {
	var listCmd tea.Cmd
	m.viewport.SetHeight(m.viewport.Height() + m.slashCommandList.Height())
	m.slashCommandList.SetHeight(0)
	m.slashCommandList.SetItems(items) // Reset items
	return m, listCmd
}
