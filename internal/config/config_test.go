package config

import (
	"os"
	"strings"
	"testing"
)

func withHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestEnsureKeyPair(t *testing.T) {
	withHome(t)

	authorized, err := EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() err = %v", err)
	}
	if !strings.HasPrefix(string(authorized), "ssh-ed25519 ") {
		t.Errorf("public half = %q, want an ssh-ed25519 authorized_keys line", authorized)
	}

	info, err := os.Stat(KeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("private key mode = %v, want 0600", perm)
	}

	private, err := os.ReadFile(KeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(private), "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Errorf("private key is not in the OpenSSH format, which is the only one ssh -i reads:\n%s", private)
	}

	again, err := EnsureKeyPair()
	if err != nil {
		t.Fatalf("second EnsureKeyPair() err = %v", err)
	}
	if string(again) != string(authorized) {
		t.Error("EnsureKeyPair() replaced an existing key, which locks the user out of every machine using it")
	}

	if err := os.Remove(PubKeyPath()); err != nil {
		t.Fatal(err)
	}
	derived, err := EnsureKeyPair()
	if err != nil {
		t.Fatalf("EnsureKeyPair() after removing the public half err = %v", err)
	}
	if string(derived) != string(authorized) {
		t.Error("a missing public half was regenerated instead of derived from the existing private key")
	}
	stored, err := os.ReadFile(PubKeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(authorized) {
		t.Errorf("%s = %q, want %q", PubKeyPath(), stored, authorized)
	}
}
