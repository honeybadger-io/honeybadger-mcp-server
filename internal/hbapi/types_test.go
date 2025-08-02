package hbapi

import (
	"encoding/json"
	"testing"
)

func TestStringOrInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "string value",
			input:   `"123"`,
			want:    "123",
			wantErr: false,
		},
		{
			name:    "integer value",
			input:   `123`,
			want:    "123",
			wantErr: false,
		},
		{
			name:    "zero integer",
			input:   `0`,
			want:    "0",
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   `-1`,
			want:    "-1",
			wantErr: false,
		},
		{
			name:    "boolean value should fail",
			input:   `true`,
			wantErr: true,
		},
		{
			name:    "object value should fail",
			input:   `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s StringOrInt
			err := json.Unmarshal([]byte(tt.input), &s)
			if (err != nil) != tt.wantErr {
				t.Errorf("StringOrInt.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(s) != tt.want {
				t.Errorf("StringOrInt.UnmarshalJSON() = %v, want %v", s, tt.want)
			}
		})
	}
}

func TestBacktraceEntry_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    BacktraceEntry
		wantErr bool
	}{
		{
			name: "number as string",
			input: `{
				"number": "42",
				"file": "/path/to/file.js",
				"method": "someMethod"
			}`,
			want: BacktraceEntry{
				Number: "42",
				File:   "/path/to/file.js",
				Method: "someMethod",
			},
			wantErr: false,
		},
		{
			name: "number as integer",
			input: `{
				"number": 42,
				"file": "/path/to/file.js",
				"method": "someMethod"
			}`,
			want: BacktraceEntry{
				Number: "42",
				File:   "/path/to/file.js",
				Method: "someMethod",
			},
			wantErr: false,
		},
		{
			name: "complete backtrace entry with source",
			input: `{
				"number": 10,
				"file": "/app/index.js",
				"method": "handleError",
				"source": {"10": "throw new Error();"},
				"context": "app"
			}`,
			want: BacktraceEntry{
				Number:  "10",
				File:    "/app/index.js",
				Method:  "handleError",
				Source:  map[string]interface{}{"10": "throw new Error();"},
				Context: "app",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entry BacktraceEntry
			err := json.Unmarshal([]byte(tt.input), &entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("BacktraceEntry.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if string(entry.Number) != string(tt.want.Number) {
					t.Errorf("BacktraceEntry.Number = %v, want %v", entry.Number, tt.want.Number)
				}
				if entry.File != tt.want.File {
					t.Errorf("BacktraceEntry.File = %v, want %v", entry.File, tt.want.File)
				}
				if entry.Method != tt.want.Method {
					t.Errorf("BacktraceEntry.Method = %v, want %v", entry.Method, tt.want.Method)
				}
				if entry.Context != tt.want.Context {
					t.Errorf("BacktraceEntry.Context = %v, want %v", entry.Context, tt.want.Context)
				}
			}
		})
	}
}
