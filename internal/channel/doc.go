// Package channel is the unified hub over messaging services
// (telegram, slack, imessage, ...).
//
// Responsibilities (per design §8.3):
//   - Define a single Channel interface with per-kind implementations
//     under internal/channel/<kind>/.
//   - Route inbound messages to sessions; dispatch outbound notifications
//     from the event bus.
//   - Add a new channel kind by implementing the interface — no router edits.
//
// Implementation lands in M4. Replaces v1's gateway/telegram, gateway/slack
// hardcoded into the router.
package channel
