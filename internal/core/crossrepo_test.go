package core

import "testing"

func TestParseCrossRepo(t *testing.T) {
	cases := []struct {
		in       string
		ok       bool
		url      string
		uuid     string
		label    string
		shortUID string
	}{
		{
			in: "#https://github.com/major0/other?track=0192f1a2-7c3e-7abc-def0-112233445566",
			ok: true, url: "https://github.com/major0/other",
			uuid:  "0192f1a2-7c3e-7abc-def0-112233445566",
			label: "other", shortUID: "0192f1a2",
		},
		{
			in: "git@github.com:major0/other.git?track=0192f1a2", ok: true,
			url: "git@github.com:major0/other.git", uuid: "0192f1a2",
			label: "other", shortUID: "0192f1a2",
		},
		{in: "#not-a-cross-repo", ok: false},
		{in: "#just-a-slug", ok: false},
		{in: "https://h/r?track=", ok: false}, // empty uuid
		{in: "?track=0192f1a2", ok: false},    // empty url
	}
	for _, c := range cases {
		ref, ok := ParseCrossRepo(c.in)
		if ok != c.ok {
			t.Errorf("ParseCrossRepo(%q) ok=%v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if ref.RepoURL != c.url || ref.UUID != c.uuid {
			t.Errorf("ParseCrossRepo(%q) = {%q, %q}, want {%q, %q}", c.in, ref.RepoURL, ref.UUID, c.url, c.uuid)
		}
		if ref.RepoLabel() != c.label {
			t.Errorf("RepoLabel(%q) = %q, want %q", c.in, ref.RepoLabel(), c.label)
		}
		if ref.ShortUUID() != c.shortUID {
			t.Errorf("ShortUUID(%q) = %q, want %q", c.in, ref.ShortUUID(), c.shortUID)
		}
	}
}
