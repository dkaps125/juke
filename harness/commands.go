package harness

var slashCommands = map[string]func(h *Harness){
	"/compact": func(h *Harness) {

	},
	"/play": func(h *Harness) {
		h.Play()
	},
	"/pause": func(h *Harness) {
		h.Pause()
	},
}

func (h *Harness) GetSlashCommands() map[string]func(h *Harness) {
	return slashCommands
}

func (h *Harness) GetSlashCommandKeys() []string {
	keys := make([]string, len(slashCommands))

	i := 0
	for k := range slashCommands {
		keys[i] = k
		i++
	}

	return keys
}
