# DEPTHS Safety Invariants

DEPTHS is an aggressive RAM reclaim tool for **macOS and Linux**. These rules are non-negotiable.

1. **Default is observe-only.** `depths scout` never sends signals.
2. **Live claims require an explicit verb** (`claim`) plus confirmation, or `--yes` after a printed dry-run plan.
3. **Hard denylist cannot be weakened by profiles** for system/session-critical processes (OS-specific, kept minimal).
4. **Never kill ancestors** of the DEPTHS process (shell, terminal, sudo parent, self).
5. **PID identity check at kill time** — pid + start time; refuse recycled PIDs.
6. **SIGTERM → grace → SIGKILL** — SIGKILL is never first unless `--force-kill` on an allowed profile.
7. **Dry-run prints exact PIDs, groups, estimates, and protected denials** before any signal.
8. **Pressure gate** — aggressive profiles refuse to run unless swap/pressure crosses the profile threshold (overridable with `--force-pressure`). Darwin uses `memory_pressure`; Linux uses PSI (`/proc/pressure/memory`) with metrics fallback.
9. **Receipt on every claim attempt** — profile, PIDs, signals, before/after pressure. Darwin: `~/Library/Logs/depths`. Linux: `$XDG_STATE_HOME/depths` or `~/.local/state/depths`.
10. **No silent bulk kill** — interactive confirm or explicit `--groups=…`.

Windows is unsupported. Estimates are labeled estimates. RSS ≠ freeable RAM.
