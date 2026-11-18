package app

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct {
	unselectedItemStyle lipgloss.Style
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	var fn func(s ...string) string
	if index == m.Index() {
		fn = func(s ...string) string {
			return fmt.Sprintf(">  %s", i)
		}
	} else {
		fn = func(s ...string) string {
			return d.unselectedItemStyle.Render(fmt.Sprintf("   %s", i))
		}
	}

	fmt.Fprint(w, fn(i.FilterValue()))
}
