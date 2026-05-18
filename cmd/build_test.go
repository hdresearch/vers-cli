package cmd

import (
	"reflect"
	"testing"
)

func TestParseBuildArgs(t *testing.T) {
	cases := []struct {
		name    string
		in      []string
		want    map[string]string
		wantErr bool
	}{
		{name: "nil", in: nil, want: nil},
		{name: "empty slice", in: []string{}, want: nil},
		{name: "single", in: []string{"FOO=bar"}, want: map[string]string{"FOO": "bar"}},
		{
			name: "multiple",
			in:   []string{"FOO=bar", "BAZ=qux"},
			want: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name: "value with equals",
			in:   []string{"URL=https://x.com/?a=1&b=2"},
			want: map[string]string{"URL": "https://x.com/?a=1&b=2"},
		},
		{
			name: "empty value",
			in:   []string{"EMPTY="},
			want: map[string]string{"EMPTY": ""},
		},
		{name: "missing equals", in: []string{"BAD"}, wantErr: true},
		{name: "one bad in list", in: []string{"OK=1", "BAD"}, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseBuildArgs(c.in)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (result=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}
