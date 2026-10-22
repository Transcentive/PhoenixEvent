package site

import "testing"

func TestResolve(t *testing.T) {
	const domain = "phoenixevent.tomsherlock.space"
	tests := []struct {
		host string
		want Address
	}{
		{"lincolnpta.phoenixevent.tomsherlock.space", Address{Organization, "lincolnpta"}},
		{"LincolnPTA.PhoenixEvent.TomSherlock.Space", Address{Organization, "lincolnpta"}},
		{"lincolnpta.phoenixevent.localhost:5174", Address{}},
		{"phoenixevent.tomsherlock.space", Address{Kind: General}},
		{"phoenixevent.tomsherlock.space.", Address{Kind: General}},
		{"a.lincolnpta.phoenixevent.tomsherlock.space", Address{}},
		{"tomsherlock.space", Address{}},
		{"evilphoenixevent.tomsherlock.space", Address{}},
		{"", Address{}},
	}
	for _, tt := range tests {
		if got := Resolve(tt.host, domain); got != tt.want {
			t.Errorf("Resolve(%q) = %+v, want %+v", tt.host, got, tt.want)
		}
	}

	if got := Resolve("lincolnpta.phoenixevent.localhost:5174", "phoenixevent.localhost"); got != (Address{Organization, "lincolnpta"}) {
		t.Errorf("development address with a port: %+v", got)
	}
}
