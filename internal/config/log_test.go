package config

import (
	"testing"

	"github.com/charmbracelet/log"
)

func TestLog_Validate(t *testing.T) {
	tests := []struct {
		name    string
		log     Log
		wantErr bool
	}{
		{
			name: "valid debug level with text format",
			log: Log{
				Level:  "debug",
				Format: "text",
			},
			wantErr: false,
		},
		{
			name: "valid info level with json format",
			log: Log{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "valid warn level with logfmt format",
			log: Log{
				Level:  "warn",
				Format: "logfmt",
			},
			wantErr: false,
		},
		{
			name: "valid error level with text format",
			log: Log{
				Level:  "error",
				Format: "text",
			},
			wantErr: false,
		},
		{
			name: "valid fatal level with json format",
			log: Log{
				Level:  "fatal",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			log: Log{
				Level:  "invalid",
				Format: "text",
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			log: Log{
				Level:  "info",
				Format: "invalid",
			},
			wantErr: true,
		},
		{
			name: "empty log level",
			log: Log{
				Level:  "",
				Format: "text",
			},
			wantErr: true,
		},
		{
			name: "empty log format",
			log: Log{
				Level:  "info",
				Format: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Log.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLog_SetLevelString(t *testing.T) {
	tests := []struct {
		name          string
		initialLevel  string
		newLevel      string
		expectedLevel string
	}{
		{
			name:          "set valid level",
			initialLevel:  "info",
			newLevel:      "debug",
			expectedLevel: "debug",
		},
		{
			name:          "set invalid level - should not change",
			initialLevel:  "info",
			newLevel:      "invalid",
			expectedLevel: "info",
		},
		{
			name:          "set empty level - should not change",
			initialLevel:  "warn",
			newLevel:      "",
			expectedLevel: "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := Log{
				Level: tt.initialLevel,
			}
			log.SetLevelString(tt.newLevel)
			if log.Level != tt.expectedLevel {
				t.Errorf("Log.SetLevelString() level = %v, want %v", log.Level, tt.expectedLevel)
			}
		})
	}
}

func TestLog_ConfigureWithLevelString(t *testing.T) {
	tests := []struct {
		name          string
		initialLevel  string
		overrideLevel string
		expectedLevel string
	}{
		{
			name:          "override with valid level",
			initialLevel:  "info",
			overrideLevel: "debug",
			expectedLevel: "debug",
		},
		{
			name:          "override with empty level - should not change",
			initialLevel:  "info",
			overrideLevel: "",
			expectedLevel: "info",
		},
		{
			name:          "override with same level - should not change",
			initialLevel:  "warn",
			overrideLevel: "warn",
			expectedLevel: "warn",
		},
		{
			name:          "override with invalid level - should not change",
			initialLevel:  "info",
			overrideLevel: "invalid",
			expectedLevel: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logConfig := Log{
				Level: tt.initialLevel,
			}
			// Parse the initial level to set ParsedLevel
			if parsedLevel, err := log.ParseLevel(tt.initialLevel); err == nil {
				logConfig.ParsedLevel = parsedLevel
			}
			logConfig.ConfigureWithLevelString(tt.overrideLevel)
			if logConfig.Level != tt.expectedLevel {
				t.Errorf("Log.ConfigureWithLevelString() level = %v, want %v", logConfig.Level, tt.expectedLevel)
			}
		})
	}
}
