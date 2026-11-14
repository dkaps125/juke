package music

import "fmt"

func msToDurationString(ms int) string {
	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds - minutes*60

	return fmt.Sprintf("%d:%02d", minutes, seconds)
}
