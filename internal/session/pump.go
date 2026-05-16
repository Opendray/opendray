package session

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// idleTailLines bounds how much of the recent stdout we ship inside
// the session.idle event when no JSONL transcript is available
// (Codex / Gemini / shell sessions). Generous on purpose — the
// channel layer (notify_snippet_max_chars, Telegram chunking) is
// the right place to clamp for any specific transport. Set high
// enough that a marathon session's last interesting screenful
// survives. The ring buffer is 1 MiB and most output averages
// ~80 chars/line, so this cap rarely bites in practice.
const idleTailLines = 1000

// pumpStdout copies bytes from the PTY into the ring buffer + fanout
// subscribers. Updates lastActivity so the idle watcher resets when
// the CLI emits output. Exits when the PTY closes (process death =>
// EOF) — at which point waitExit handles cleanup.
func (m *Manager) pumpStdout(rs *runningSession) {
	defer m.wg.Done()
	buf := make([]byte, pumpBufSize)
	for {
		n, err := rs.pty.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			_, _ = rs.ring.Write(chunk)
			if rs.vt != nil {
				// Feed the virtual terminal so RecentScreen sees the
				// post-ANSI state. Errors here are non-fatal — a
				// malformed escape would just leave a glyph somewhere;
				// the live xterm.js client still sees the raw bytes
				// via fanout.
				_, _ = rs.vt.Write(chunk)
			}
			rs.fanout(chunk)
			if rs.markActive(time.Now()) {
				m.flipBackToRunning(rs)
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				m.log.Debug("pty read closed", "session", rs.sess.ID, "err", err)
			}
			return
		}
	}
}

// idleWatcher polls the session's lastActivity at idleInterval cadence
// and fires session.idle once per active→idle window. Exits when the
// session ends.
func (m *Manager) idleWatcher(rs *runningSession) {
	defer m.wg.Done()
	ticker := time.NewTicker(m.idleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-rs.endedCh:
			return
		case now := <-ticker.C:
			if !rs.checkIdle(now, m.idleThreshold) {
				continue
			}
			rs.sessMu.Lock()
			terminal := rs.sess.State.IsTerminal()
			if !terminal {
				rs.sess.State = StateIdle
			}
			rs.sessMu.Unlock()
			if terminal {
				return
			}
			data := map[string]any{
				"session_id":  rs.sess.ID,
				"idle_for_ms": m.idleThreshold.Milliseconds(),
			}
			// Snippet source priority:
			//   1. Claude JSONL transcript (claude provider)
			//   2. Gemini logs.json transcript (gemini provider)
			//   3. Virtual-terminal screen snapshot — what the user
			//      sees in the live web terminal right now
			//   4. Raw ring-buffer tail — defensive fallback
			//
			// We deliberately do NOT cap byte / line counts at the
			// source. The channel layer owns user-facing truncation
			// (notify_snippet_max_chars + per-platform chunking, e.g.
			// Telegram's 3800-rune splitter). Capping here means the
			// operator-facing "Unlimited — split into messages"
			// option silently loses content; let the source emit
			// everything and let the channel decide.
			snippet := ""
			rs.sessMu.RLock()
			provider := rs.sess.ProviderID
			cwd := rs.sess.Cwd
			rs.sessMu.RUnlock()
			if provider == "claude" && cwd != "" {
				snippet = claudeRecentResponse(cwd)
			}
			if snippet == "" && provider == "gemini" && cwd != "" {
				snippet = geminiRecentResponse(cwd)
			}
			if snippet == "" && rs.vt != nil {
				snippet = ScreenSnapshot(rs.vt)
				// Screen snapshots still need TUI chrome stripping
				// (model bar, bypass-permissions hint, separator runs,
				// status spinners). JSONL output is already clean.
				snippet = FilterClaudeChrome(snippet)
			}
			if snippet == "" {
				snippet = CleanTerminalOutput(string(rs.ring.Snapshot()), idleTailLines)
				snippet = FilterClaudeChrome(snippet)
			}
			if snippet != "" {
				data["recent_output"] = snippet
			}
			m.bus.Publish(eventbus.Event{
				Topic: "session.idle",
				Data:  data,
			})
		}
	}
}

// flipBackToRunning is called when activity arrives during an idle
// window. Idempotent.
func (m *Manager) flipBackToRunning(rs *runningSession) {
	rs.sessMu.Lock()
	defer rs.sessMu.Unlock()
	if rs.sess.State == StateIdle {
		rs.sess.State = StateRunning
	}
}

// waitExit blocks on cmd.Wait() and runs end-of-life cleanup exactly
// once per session: persist exit_code, close PTY + subscribers, fire
// session.ended.
func (m *Manager) waitExit(rs *runningSession) {
	defer m.wg.Done()
	err := rs.cmd.Wait()
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	rs.endOnce.Do(func() {
		// Distinguish user-initiated stops (Manager.Stop set
		// stopRequested) from spontaneous exits. The DB row
		// persists either way so the user can Restart.
		state := StateEnded
		if m.consumeStopRequest(rs.sess.ID) {
			state = StateStopped
		}

		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.store.MarkTerminal(dbCtx, rs.sess.ID, state, exitCode); err != nil {
			m.log.Error("mark session terminal", "session", rs.sess.ID, "err", err)
		}

		now := time.Now().UTC()
		ec := exitCode
		rs.sessMu.Lock()
		rs.sess.State = state
		rs.sess.EndedAt = &now
		rs.sess.ExitCode = &ec
		rs.sessMu.Unlock()

		_ = rs.pty.Close()
		rs.closeSubs()
		if rs.tempDir != "" {
			_ = os.RemoveAll(rs.tempDir)
		}
		close(rs.endedCh)

		// Drop from the live map so a subsequent Start() can
		// install a fresh runningSession under the same id.
		m.mu.Lock()
		if cur, ok := m.sessions[rs.sess.ID]; ok && cur == rs {
			delete(m.sessions, rs.sess.ID)
		}
		m.mu.Unlock()

		topic := "session.ended"
		if state == StateStopped {
			topic = "session.stopped"
		}
		m.bus.Publish(eventbus.Event{
			Topic: topic,
			Data: map[string]any{
				"session_id": rs.sess.ID,
				"exit_code":  exitCode,
				"ended_at":   now,
				"state":      string(state),
			},
		})
	})
}

// fanout broadcasts a stdout chunk to all live subscribers. Slow
// subscribers (full buffer) drop this chunk — same backpressure policy
// as eventbus.Hub.
func (rs *runningSession) fanout(p []byte) {
	rs.subsMu.Lock()
	defer rs.subsMu.Unlock()
	for ch := range rs.subs {
		select {
		case ch <- p:
		default:
		}
	}
}

func (rs *runningSession) closeSubs() {
	rs.subsMu.Lock()
	defer rs.subsMu.Unlock()
	for ch := range rs.subs {
		close(ch)
		delete(rs.subs, ch)
	}
}
