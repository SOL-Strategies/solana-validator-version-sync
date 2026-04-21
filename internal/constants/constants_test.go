package constants

import "testing"

func TestNormalizeClientName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "canonical rakurai client remains unchanged",
			input:  ClientNameRakurai,
			output: ClientNameRakurai,
		},
		{
			name:   "legacy rakurai alias normalizes to canonical name",
			input:  "rakurai",
			output: ClientNameRakurai,
		},
		{
			name:   "other client names are unchanged",
			input:  ClientNameAgave,
			output: ClientNameAgave,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeClientName(tt.input); got != tt.output {
				t.Fatalf("NormalizeClientName(%q) = %q, want %q", tt.input, got, tt.output)
			}
		})
	}
}

func TestValidateClientName(t *testing.T) {
	tests := []struct {
		name      string
		client    string
		wantError bool
	}{
		{
			name:      "accepts canonical rakurai client name",
			client:    ClientNameRakurai,
			wantError: false,
		},
		{
			name:      "accepts legacy rakurai alias",
			client:    "rakurai",
			wantError: false,
		},
		{
			name:      "rejects unknown client name",
			client:    "invalid-client",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientName(tt.client)
			if (err != nil) != tt.wantError {
				t.Fatalf("ValidateClientName(%q) error = %v, wantError %v", tt.client, err, tt.wantError)
			}
		})
	}
}
