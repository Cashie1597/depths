# DEPTHS

Standalone macOS terminal tool for reclaiming RAM under swap pressure.

```bash
cd ~/PROJECTS/03-labs/depths
go build -o depths ./cmd/depths
./depths                 # starter console (ASCII + menu)
./depths scout           # live watch
./depths claim --dry-run # plan only
```

Part of the **Cashie Relay** project family. Clean-room · MIT · no Mole source.

Read [SAFETY.md](SAFETY.md) before live claim.
