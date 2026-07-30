# DEPTHS

macOS + Linux terminal tool for observing memory pressure and planning safe reclaim.

**Landing:** https://cashie1597.github.io/depths/  
**Repo:** https://github.com/Cashie1597/depths

```bash
git clone https://github.com/Cashie1597/depths.git
cd depths
go build -o depths ./cmd/depths
./depths                 # starter console (status before actions)
./depths scout           # live watch
./depths claim --dry-run # plan only
```

**Supported:** macOS (Darwin), Linux (Fedora / systemd hosts)  
**Unsupported:** Windows (explicit error — no half-broken claim path)

Part of the **Cashie Relay** project family. Clean-room · MIT · no Mole source.

DEPTHS is a pressure observer + safe reclaim planner — not a generic system cleaner.

Read [SAFETY.md](SAFETY.md) before live claim.

## Smoke checklist (Fedora)

```bash
go test ./...
go build -o depths ./cmd/depths
./depths scout --plain --once
./depths claim --dry-run --force-pressure
```

Expect: PSI or metrics-based pressure, swap fields, denylist holding `systemd` / `sshd`, receipts under `~/.local/state/depths` after a live claim.
