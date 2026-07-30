# DEPTHS

Standalone macOS terminal tool for reclaiming RAM under swap pressure.

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

Part of the **Cashie Relay** project family. Clean-room · MIT · no Mole source.

Read [SAFETY.md](SAFETY.md) before live claim.
