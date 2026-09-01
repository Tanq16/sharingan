package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	u "github.com/tanq16/sharingan/utils"
	"golang.org/x/crypto/ssh"
)

func KeyPath() string { return filepath.Join(u.Dir(), "id_ed25519") }

func PubKeyPath() string { return filepath.Join(u.Dir(), "id_ed25519.pub") }

func KnownHostsPath() string { return filepath.Join(u.Dir(), "known_hosts") }

func EnsureKeyPair() ([]byte, error) {
	if err := u.EnsureDir(); err != nil {
		return nil, err
	}
	priv, err := loadPrivateKey()
	if err != nil {
		return nil, err
	}
	if priv == nil {
		if priv, err = generatePrivateKey(); err != nil {
			return nil, err
		}
	}

	pub, err := ssh.NewPublicKey(priv.Public())
	if err != nil {
		return nil, err
	}
	authorized := ssh.MarshalAuthorizedKey(pub)

	if _, err := os.Stat(PubKeyPath()); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(PubKeyPath(), authorized, 0o600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return authorized, nil
}

// An existing key is reused, never replaced: regenerating locks the user out of every live machine.
func loadPrivateKey() (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(KeyPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	parsed, err := ssh.ParseRawPrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", KeyPath(), err)
	}
	switch key := parsed.(type) {
	case *ed25519.PrivateKey:
		return *key, nil
	case ed25519.PrivateKey:
		return key, nil
	default:
		return nil, fmt.Errorf("%s holds a %T, want an ed25519 key", KeyPath(), parsed)
	}
}

func generatePrivateKey() (ed25519.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	// ssh -i rejects a PKCS#8 ed25519 key, and the standard library encodes no other format.
	block, err := ssh.MarshalPrivateKey(priv, "sharingan")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(KeyPath(), pem.EncodeToMemory(block), 0o600); err != nil {
		return nil, err
	}
	return priv, nil
}
