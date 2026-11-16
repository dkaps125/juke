package harness

type Prompt struct {
	Prompt     string
	TimeToWait string
}

type TaskID struct {
	ID string
}

type ToolSource interface {
	// Schedules a task for the future. Takes the prompt to execute and the time to wait before running it. TimeToWait should be passed as a well-formatted duration string, like 1h, 30m, 20s.
	ScheduleTask(Prompt) TaskStartResult
	// Deletes an already scheduled task
	DeleteTask(TaskID) bool
}
