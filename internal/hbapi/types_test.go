package hbapi

import (
	"encoding/json"
	"testing"
)

func TestNumber_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "string value",
			input:   `"123"`,
			want:    123,
			wantErr: false,
		},
		{
			name:    "integer value",
			input:   `123`,
			want:    123,
			wantErr: false,
		},
		{
			name:    "zero integer",
			input:   `0`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   `-1`,
			want:    -1,
			wantErr: false,
		},
		{
			name:    "string negative integer",
			input:   `"-42"`,
			want:    -42,
			wantErr: false,
		},
		{
			name:    "non-numeric string should fail",
			input:   `"abc"`,
			wantErr: true,
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
			var n Number
			err := json.Unmarshal([]byte(tt.input), &n)
			if (err != nil) != tt.wantErr {
				t.Errorf("Number.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && int(n) != tt.want {
				t.Errorf("Number.UnmarshalJSON() = %v, want %v", n, tt.want)
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
				Number: 42,
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
				Number: 42,
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
				Number:  10,
				File:    "/app/index.js",
				Method:  "handleError",
				Source:  map[string]interface{}{"10": "throw new Error();"},
				Context: "app",
			},
			wantErr: false,
		},
		{
			name: "backtrace entry with all optional fields",
			input: `{
				"number": "25",
				"column": 15,
				"file": "/app/models/user.rb",
				"method": "authenticate",
				"class": "User",
				"type": "instance",
				"args": ["email@example.com", "password"],
				"source": {"25": "user = User.find_by(email: email)"},
				"context": "app"
			}`,
			want: BacktraceEntry{
				Number:  25,
				Column:  numberPtr(15),
				File:    "/app/models/user.rb",
				Method:  "authenticate",
				Class:   "User",
				Type:    "instance",
				Args:    []interface{}{"email@example.com", "password"},
				Source:  map[string]interface{}{"25": "user = User.find_by(email: email)"},
				Context: "app",
			},
			wantErr: false,
		},
		{
			name: "backtrace entry with column as string",
			input: `{
				"number": 42,
				"column": "8",
				"file": "/lib/helper.js",
				"method": "processData"
			}`,
			want: BacktraceEntry{
				Number: 42,
				Column: numberPtr(8),
				File:   "/lib/helper.js",
				Method: "processData",
			},
			wantErr: false,
		},
		{
			name: "number as string negative",
			input: `{
				"number": "-10",
				"file": "/app/test.rb",
				"method": "test"
			}`,
			want: BacktraceEntry{
				Number: -10,
				File:   "/app/test.rb",
				Method: "test",
			},
			wantErr: false,
		},
		{
			name: "invalid string number",
			input: `{
				"number": "abc",
				"file": "/app/test.rb",
				"method": "test"
			}`,
			wantErr: true,
		},
		{
			name: "number as boolean should fail",
			input: `{
				"number": true,
				"file": "/app/test.rb",
				"method": "test"
			}`,
			wantErr: true,
		},
		{
			name: "column as boolean should fail",
			input: `{
				"number": 1,
				"column": true,
				"file": "/app/test.rb",
				"method": "test"
			}`,
			wantErr: true,
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
				if int(entry.Number) != int(tt.want.Number) {
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
				// Check Column
				if tt.want.Column != nil && entry.Column != nil {
					if int(*entry.Column) != int(*tt.want.Column) {
						t.Errorf("BacktraceEntry.Column = %v, want %v", *entry.Column, *tt.want.Column)
					}
				} else if (tt.want.Column == nil) != (entry.Column == nil) {
					t.Errorf("BacktraceEntry.Column = %v, want %v", entry.Column, tt.want.Column)
				}
				// Check Class
				if entry.Class != tt.want.Class {
					t.Errorf("BacktraceEntry.Class = %v, want %v", entry.Class, tt.want.Class)
				}
				// Check Type
				if entry.Type != tt.want.Type {
					t.Errorf("BacktraceEntry.Type = %v, want %v", entry.Type, tt.want.Type)
				}
				// Check Args length
				if len(entry.Args) != len(tt.want.Args) {
					t.Errorf("BacktraceEntry.Args length = %v, want %v", len(entry.Args), len(tt.want.Args))
				}
			}
		})
	}
}

// Helper function to create Number pointers
func numberPtr(i int) *Number {
	n := Number(i)
	return &n
}