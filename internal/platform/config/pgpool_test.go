package config

import "testing"

func TestPGMaxConns(t *testing.T) {
	cases := []struct {
		env  string
		want int32
	}{
		{"", 4},
		{"8", 8},
		{" 12 ", 12},
		{"0", 4},
		{"-3", 4},
		{"lots", 4},
		{"99999999999", 4},
	}
	for _, c := range cases {
		t.Setenv("PG_MAX_CONNS", c.env)
		if got := PGMaxConns(DefaultPGMaxConns); got != c.want {
			t.Errorf("PG_MAX_CONNS=%q: got %d, want %d", c.env, got, c.want)
		}
	}
	if DefaultPGMaxConns != 4 {
		t.Fatalf("DefaultPGMaxConns = %d, L21 pins 4", DefaultPGMaxConns)
	}
}
