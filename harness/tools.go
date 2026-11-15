package harness

import "time"

func (h *Harness) ScheduleTask(prompt Prompt) bool {
	duration, _ := time.ParseDuration(prompt.TimeToWait)
	timer := time.After(duration)

	go func() {
		<-timer
		h.Prompt(prompt.Prompt)
	}()

	return true
}
