package harness

import (
	"crypto/rand"
	"time"
)

type TaskStartResult struct {
	ID string
}

type Task struct {
	ID         string
	cancelChan chan struct{}
}

func (h *Harness) ScheduleTask(prompt Prompt) TaskStartResult {
	duration, _ := time.ParseDuration(prompt.TimeToWait)
	timer := time.After(duration)
	cancelChan := make(chan struct{})

	go func() {
		select {
		case <-timer:
			h.Prompt(prompt.Prompt)
		case <-cancelChan:
			return
		}
	}()

	taskID := rand.Text()[:7]
	h.tasks[taskID] = Task{
		ID:         taskID,
		cancelChan: cancelChan,
	}

	return TaskStartResult{
		ID: taskID,
	}
}

func (h *Harness) DeleteTask(taskID TaskID) bool {
	if task, ok := h.tasks[taskID.ID]; ok {
		task.cancelChan <- struct{}{}
		delete(h.tasks, taskID.ID)

		return true
	}

	return false
}
