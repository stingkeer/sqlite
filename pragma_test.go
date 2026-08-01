package sqlite

import (
	"strings"
	"testing"
)

func TestApplyDefaultPragmas(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want []string // substrings that must appear
		skip []string // substrings that must NOT appear (beyond the first)
	}{
		{
			name: "empty dsn gets all defaults",
			dsn:  "",
			want: []string{"?_pragma=journal_mode(WAL)", "_pragma=synchronous(NORMAL)", "_pragma=busy_timeout(5000)"},
		},
		{
			name: "plain file path",
			dsn:  "test.db",
			want: []string{"test.db?_pragma=journal_mode(WAL)", "_pragma=synchronous(NORMAL)", "_pragma=busy_timeout(5000)"},
		},
		{
			name: "existing query uses ampersand separator",
			dsn:  "test.db?_pragma=foreign_keys(1)",
			want: []string{"&_pragma=journal_mode(WAL)", "_pragma=synchronous(NORMAL)", "_pragma=busy_timeout(5000)"},
		},
		{
			name: "user journal_mode preserved, others added",
			dsn:  "test.db?_pragma=journal_mode(DELETE)",
			want: []string{"_pragma=journal_mode(DELETE)", "_pragma=synchronous(NORMAL)", "_pragma=busy_timeout(5000)"},
			skip: []string{"journal_mode(WAL)"},
		},
		{
			name: "case-insensitive detection",
			dsn:  "test.db?_pragma=JOURNAL_MODE(WAL)",
			want: []string{"_pragma=JOURNAL_MODE(WAL)", "_pragma=synchronous(NORMAL)", "_pragma=busy_timeout(5000)"},
			skip: []string{"journal_mode(WAL)"},
		},
		{
			name: "all set => unchanged",
			dsn:  "test.db?_pragma=journal_mode(WAL)&_pragma=synchronous(OFF)&_pragma=busy_timeout(100)",
			want: []string{"_pragma=journal_mode(WAL)", "_pragma=synchronous(OFF)", "_pragma=busy_timeout(100)"},
			skip: []string{"synchronous(NORMAL)", "busy_timeout(5000)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyDefaultPragmas(tt.dsn)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("expected %q to contain %q", got, w)
				}
			}
			for _, s := range tt.skip {
				if strings.Contains(got, s) {
					t.Errorf("expected %q NOT to contain %q", got, s)
				}
			}
		})
	}
}
