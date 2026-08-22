<div align="center">
  <img src=".github/assets/logo.svg" alt="sharingan Logo" width="200">
  <h1>sharingan</h1>

  <a href="https://github.com/tanq16/sharingan/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/sharingan/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/sharingan/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/sharingan"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#install">Install</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#notes">Notes</a>
</div>

---

sharingan is a single Go binary that manages long-lived EC2 workstations. It builds one set of network scaffolding per AWS account and region, launches named machines into it, and connects to them over SSH.

It exists so a remote dev box is one command away and costs nothing but disk while it is stopped. It is not a fleet manager, it is AWS only, and SSH is the only way in.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Configuration | `set-config` | Choose the active AWS profile and region |
| Scaffolding | `setup`, `teardown` | Create and delete the per-region VPC, subnet, gateway, routing, security group, and key pair |
| Machines | `create`, `start`, `stop`, `modify`, `rm` | Launch a machine, run it, park it, resize it, destroy it |
| Access | `ssh`, `status` | Connect to a machine and report its bootstrap progress |
| Reporting | `show`, `costs` | Inspect what exists and what it costs |

## Install

Download the binary for your platform from the [releases page](https://github.com/tanq16/sharingan/releases), then make it executable and put it on your `PATH`. Builds are published for `linux-amd64`, `linux-arm64`, `darwin-amd64`, and `darwin-arm64`.

```bash
curl -fLo sharingan https://github.com/tanq16/sharingan/releases/latest/download/sharingan-darwin-arm64
chmod +x sharingan && mv sharingan /usr/local/bin/
```

Building from source needs Go 1.26.5 or newer:

```bash
git clone https://github.com/tanq16/sharingan && cd sharingan && make build
```

## Usage

Every command except `set-config` reads the active profile and region from `~/.config/sharingan/config.json` and exits non-zero when it is missing. `--debug` turns on verbose logging and `--for-ai` switches to plain text output with piped input, for driving the tool from a script or an agent.

```bash
sharingan set-config --profile my-profile --region us-east-2
sharingan setup
sharingan create devbox
sharingan ssh devbox
```

| Command | Behaviour |
|---|---|
| `set-config [--profile P] [--region R]` | Writes `config.json`. With no flags it prompts for both. There is no default profile and no default region. |
| `setup` | Creates whatever scaffolding the region is missing. Idempotent, so it is safe to rerun. |
| `teardown` | Deletes the scaffolding. Refuses while any managed instance still exists. |
| `create <name>` | Prompts for size, architecture, and disk, then launches Ubuntu 26.04 with Docker, zsh, Homebrew, and `cps` installed by cloud-init. |
| `start <name>` / `stop <name>` | Starts or stops the machine and waits for it to settle. |
| `modify <name>` | Changes vCPU and RAM on the same architecture. Disk size cannot change. |
| `rm <name>` | Terminates the machine and its root volume. Requires the name retyped, or `--yes`. |
| `ssh <name> [args...]` | Execs the system `ssh`, so `ssh_config`, agents, and `ProxyJump` all still apply. Trailing arguments pass through. |
| `status <name>` | Reports cloud-init and bootstrap progress over SSH. |
| `show [--origin]` | Table of local state, or the same table rebuilt from live API queries. |
| `costs` | Table of every size, architecture, class, and disk combination with its monthly price. |

Sizes are offered as fourteen vCPU and RAM shapes across two classes, burstable and dedicated, on `x86_64` and `arm64`. The concrete instance type is resolved live: the cheapest modern-family type that is sold in the active region and matches the shape exactly. Disk size is independent of the shape, from 50 GB to 500 GB.

### Configuration

Everything lives under `~/.config/sharingan`, hardcoded, with no environment variable override.

| File | Holds |
|---|---|
| `config.json` | The active profile and region, written only by `set-config` |
| `state.json` | Per account and region: the scaffolding ids and the machines |
| `id_ed25519`, `id_ed25519.pub` | The one key pair every machine authorizes |
| `known_hosts` | Host keys, scoped to this tool rather than your own |

## Notes

- **Prices are read live and never shipped.** `costs` and the `create` selector price the region through the AWS Price List API on every run. A failed lookup prints the failure and no numbers.
- **arm64 is cheaper at an identical shape**, by roughly 11% burstable and 19% dedicated, and nothing in the bootstrap is x86-only.
- **A stopped machine costs its disk and nothing else.** The public IPv4 address is released on stop, so the address changes on every start. `ssh` pins the host key to the machine name rather than the address, so `known_hosts` stays clean.
- **`state.json` is a cache, never the source of truth.** Deleting it is recoverable, because every command rediscovers resources by their `ManagedBy=sharingan` tag.
- **Ports 22 and 443 are open to `0.0.0.0/0`.** SSH is key-only, but whatever you later run behind 443 is reachable by anyone and its own authentication is the only control.
- **One key means one blast radius.** There is no Session Manager fallback, so losing `id_ed25519` or breaking `sshd` leaves only two ways back in: attach the root volume to another instance, or `rm` the machine.
- **Disk size is fixed for the life of a machine**, and instances created by other tooling are never adopted.
