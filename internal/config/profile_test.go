package config

import "testing"

func TestCreateProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, bad := range []string{"", "default", "Brücke", "a/b", "-x"} {
		if CreateProfile(bad) == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
	if err := CreateProfile("bruecke"); err != nil {
		t.Fatal(err)
	}
	if CreateProfile("bruecke") == nil {
		t.Error("duplicate should fail")
	}
	if names, _ := ListProfiles(); len(names) != 1 || names[0] != "bruecke" {
		t.Errorf("ListProfiles = %v", names)
	}
}
