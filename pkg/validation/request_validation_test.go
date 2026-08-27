package validation

import "testing"

func TestParseUint64ID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{name: "plain id", input: "42", want: 42},
		{name: "surrounding whitespace is trimmed", input: "  42  ", want: 42},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "not a number", input: "abc", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "zero is not a valid id", input: "0", wantErr: true},
		{name: "overflows uint64", input: "99999999999999999999999", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, appErr := ParseUint64ID(tc.input)

			if tc.wantErr {
				if appErr == nil {
					t.Fatalf("ParseUint64ID(%q) = %d, want an error", tc.input, got)
				}
				return
			}

			if appErr != nil {
				t.Fatalf("ParseUint64ID(%q) returned %v, want no error", tc.input, appErr)
			}
			if got != tc.want {
				t.Errorf("ParseUint64ID(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestRequireNonEmpty(t *testing.T) {
	if got, appErr := RequireNonEmpty("  value  "); appErr != nil || got != "value" {
		t.Errorf("RequireNonEmpty = (%q, %v), want (\"value\", nil)", got, appErr)
	}

	if _, appErr := RequireNonEmpty("   "); appErr == nil {
		t.Error("RequireNonEmpty accepted a whitespace-only value, want an error")
	}
}

func TestNormalizedEnum(t *testing.T) {
	if got := NormalizedEnum("  APPROVED "); got != "approved" {
		t.Errorf("NormalizedEnum = %q, want %q", got, "approved")
	}
}

func TestTrimmedString(t *testing.T) {
	if got := TrimmedString("\t hello \n"); got != "hello" {
		t.Errorf("TrimmedString = %q, want %q", got, "hello")
	}
}
