package handlers

import "testing"

func TestParseSeekArgument(t *testing.T) {
	tests := []struct {
		input    string
		seconds  int
		absolute bool
		wantErr  bool
	}{
		{input: "158", seconds: 158, absolute: false},
		{input: "25:00", seconds: 1500, absolute: true},
		{input: "1:02:30", seconds: 3750, absolute: true},
		{input: "1:60", wantErr: true},
		{input: "1:2:3:4", wantErr: true},
		{input: "not-a-time", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			seconds, absolute, err := parseSeekArgument(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSeekArgument(%q): %v", tt.input, err)
			}
			if seconds != tt.seconds || absolute != tt.absolute {
				t.Fatalf("got (%d, %t), want (%d, %t)", seconds, absolute, tt.seconds, tt.absolute)
			}
		})
	}
}
