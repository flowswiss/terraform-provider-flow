package flow

import "testing"

func TestNormalizeSecondsDuration(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "10s", want: "10s"},
		{in: "60s", want: "1m0s"},
		{in: "1m", want: "1m0s"},
		{in: "2m30s", want: "2m30s"},
		{in: "1h", want: "1h0m0s"},
		{in: "1500ms", want: "1s"},
		{in: "500ms", wantErr: true},
		{in: "0s", wantErr: true},
		{in: "ten", wantErr: true},
	} {
		got, err := normalizeSecondsDuration(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("normalizeSecondsDuration(%q) = %q, want an error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeSecondsDuration(%q) returned %s", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("normalizeSecondsDuration(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
