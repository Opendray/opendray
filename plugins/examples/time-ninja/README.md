# time-ninja

The OpenDray Plugin Platform M1 reference plugin. Exercises all four declarative
contribution points (commands, statusBar, keybindings, menus) and one capability
posture (empty permissions — no risky grants).

## Try it

1. Start OpenDray with `OPENDRAY_ALLOW_LOCAL_PLUGINS=1 opendray`.
2. `opendray plugin install ./plugins/examples/time-ninja`.
3. Confirm the empty-permissions consent screen.
4. Press `Ctrl+Alt+P` (or `Cmd+Alt+P` on Mac), or tap the 🍅 chip in the status bar.
5. You should see a "Pomodoro started — 25 minutes" toast.

## What this plugin proves

- Install flow works with zero dangerous caps (consent screen shows a harmless list).
- Contribution points round-trip through the manifest parser + registry + HTTP API.
- The command dispatcher's `notify` run-kind is functional.
- Keybinding, status-bar, menu all fire the same command id.
- Uninstall leaves no trace.
