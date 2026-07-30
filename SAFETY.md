# DEPTHS Safety Invariants

DEPTHS is an aggressive RAM reclaim tool. These rules are non-negotiable.

1. **Default is observe-only.** `depths scout` never sends signals.
2. **Live claims require an explicit verb** (`claim`) plus confirmation, or `--yes` after a printed dry-run plan.
3. **Hard denylist cannot be weakened by profiles** for system/session-critical processes.
4. **Never kill ancestors** of the DEPTHS process (shell, terminal, sudo parent, self).
5. **PID identity check at kill time** — pid + start time; refuse recycled PIDs.
6. **SIGTERM → grace → SIGKILL** — SIGKILL is never first unless `--force-kill` on an allowed profile.
7. **Dry-run prints exact PIDs, groups, estimates, and protected denials** before any signal.
8. **Pressure gate** — aggressive profiles refuse to run unless swap/pressure crosses the profile threshold (overridable with `--force-pressure`).
9. **Receipt on every claim attempt** — profile, PIDs, signals, before/after pressure.
10. **No silent bulk kill** — interactive confirm or explicit `--groups=…`.

Estimates are labeled estimates. RSS ≠ freeable RAM.
