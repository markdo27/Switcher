# Switcher on Windows (experimental fork)

This fork adds a native Windows amd64 dashboard and command line, without WSL.
It is a local test build based on upstream 1.40.0, not an upstream Windows release.

## Install

Download/extract the Windows preview archive and run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\install-windows.ps1 -Start
```

The executable is installed in `%LOCALAPPDATA%\Switcher`, with a Start menu
shortcut. It does not require administrator rights, change PATH, start at login,
import credentials, or reconfigure running Claude/Codex sessions.
Open http://127.0.0.1:8787 and sign in to add separate accounts.
For Codex, choose the account and run `switcher.exe setup codex` explicitly when
ready to route future requests. Do not reconfigure an active agent session.

From source, install Go 1.27.1+ and run the same script. It builds the executable
when no adjacent `switcher.exe` is present. Use `-Destination` for another folder.

## Implemented

- Native Windows executable, hidden detached server and browser URL opening.
- Original dashboard, OAuth flows and HTTP provider proxies. Live provider
  authentication and billing have not been tested with real accounts.
- Account/state and settings storage with Windows DACL checks. Other principals,
  reparse-point destinations, hard-linked credential files and oversized files
  are refused. Newly written secrets receive a restrictive DACL at creation.
- TCP owner lookup using the Windows TCP table and process-token SID, for both
  IPv4 and IPv6, including the cross-process self-test.
- Codex configuration transactions, byte-range locking, backups, conflict
  detection and restoration, tested only against synthetic temporary profiles.
- `SWITCHER_DATA_DIR` selects an absolute isolated Switcher profile for testing.

## Not available in this preview

- Native Claude Code credential switching/import and session synchronization.
  Claude proxy accounts are independent of Claude Code's existing login.
- Claude Desktop relay/routing and conversation metadata association.
- System tray, Start at login, keep awake, system-load telemetry.
- Automatic updates/restart, LAN/TLS listener changes and a Windows Tailscale
  helper. These features must not be inferred from the upstream macOS UI.

Automatic binary installation and listener changes return explicit unsupported
errors. This fork retains the upstream UI; some labels still describe Macs or
Linux. Sharing/phone workflows have not been validated on Windows.

## Verification

```powershell
go build .
go test ./internal/winfs ./internal/codexcfg ./internal/localpeer ./internal/store ./internal/provider/claude ./internal/server ./internal/cli -run '^TestWindows' -count=1
go vet ./internal/winfs ./internal/codexcfg ./internal/localpeer ./internal/store ./internal/settings ./internal/config
```

The upstream `go test ./...` suite is not green on Windows: several tests import
Unix syscalls, assume Unix file modes or `$HOME`, or exercise intentionally
unsupported native Claude/Desktop features. They have not been hidden or marked
passing. Selected portable proxy/login tests and live local HTTP smoke checks
are run separately. This is an experimental port, not full platform parity.

Stop `switcher.exe` in Task Manager, then remove the installed executable folder
and the Start menu shortcut to uninstall. Account data stays in
`%USERPROFILE%\.switcher`; preserve it unless you intend to delete your logins.
