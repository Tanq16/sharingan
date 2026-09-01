package machine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/tanq16/sharingan/internal/awsx"
	"github.com/tanq16/sharingan/internal/config"
)

const probeCommand = "cloud-init status; test -f /home/ubuntu/.sharingan_bootstrap_done && echo bootstrap-done || echo bootstrap-pending"

type SSHConfig struct {
	Name string
	Args []string
}

// The running process is replaced, so this returns only when the handover fails.
func SSH(ctx context.Context, c *awsx.Clients, cfg SSHConfig) error {
	ip, err := connectIP(ctx, c, cfg.Name)
	if err != nil {
		return err
	}
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return err
	}
	args := sshArgs(cfg.Name, ip, config.KeyPath(), config.KnownHostsPath(), cfg.Args)
	return syscall.Exec(sshPath, args, os.Environ())
}

type StatusConfig struct {
	Name string
}

func Status(ctx context.Context, c *awsx.Clients, cfg StatusConfig) (string, error) {
	ip, err := connectIP(ctx, c, cfg.Name)
	if err != nil {
		return "", err
	}
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return "", err
	}
	args := sshArgs(cfg.Name, ip, config.KeyPath(), config.KnownHostsPath(), []string{probeCommand})

	probe := exec.CommandContext(ctx, sshPath, args[1:]...)
	var stderr strings.Builder
	probe.Stderr = &stderr
	out, err := probe.Output()
	if err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return "", fmt.Errorf("%s: %w", detail, err)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HostKeyAlias pins the host key to the machine name, because the public IP changes on every start.
func sshArgs(name, ip, keyPath, knownHostsPath string, passthrough []string) []string {
	args := []string{"ssh",
		"-i", keyPath,
		"-o", "UserKnownHostsFile=" + knownHostsPath,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "HostKeyAlias=sharingan-" + name,
		sshUser + "@" + ip,
	}
	return append(args, passthrough...)
}

func connectIP(ctx context.Context, c *awsx.Clients, name string) (string, error) {
	inst, err := requireInstance(ctx, c, name)
	if err != nil {
		return "", err
	}
	if inst.PublicIP == "" {
		return "", fmt.Errorf("%s is %s and has no public address", name, inst.State)
	}
	return inst.PublicIP, nil
}
