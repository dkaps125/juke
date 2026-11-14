package music

import "testing"

func TestMsToDurationString(t *testing.T) {
	tests := []struct {
		in  int
		out string
	}{
		{1000, "0:01"},
		{60000, "1:00"},
		{71000, "1:11"},
		{1211000, "20:11"},
	}

	for _, tt := range tests {
		t.Run(tt.out, func(t *testing.T) {
			got := msToDurationString(tt.in)
			if got != tt.out {
				t.Errorf("got %s want %s", got, tt.out)
			}
		})
	}
}
