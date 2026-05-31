package session

import (
	"regexp"
	"strings"
	"time"
)

// rateLimitBannerRe matches the Claude CLI's "session limit reached"
// banner. Observed forms (capture from real CLI output):
//
//	You've hit your session limit · resets 10:20am (UTC)
//	You've hit your session limit · resets 10:20 (UTC)
//	You've hit your session limit · resets 22:30 (UTC)
//	you have hit your session limit · resets 9:00am (UTC)
//
// The middle separator is U+00B7 (·) in observed output but we accept
// '-' or '|' as fallbacks against future copy changes. Time is
// captured as HH or HH:MM optionally followed by am/pm, case
// insensitive, in (UTC). Anything that doesn't match exactly is
// ignored — we'd rather miss a banner than auto-switch on a
// false positive.
var rateLimitBannerRe = regexp.MustCompile(
	`(?i)you'?ve\s+hit\s+your\s+session\s+limit\s*[·\-|]+\s*resets?\s+(\d{1,2}(?::\d{2})?\s*(?:am|pm)?)\s*\(\s*UTC\s*\)`,
)

// ScanForRateLimitBanner inspects window for the Claude rate-limit
// banner. Returns (resetTime, true) on match — resetTime is in UTC,
// today's date if the time-of-day is still in the future relative to
// 'now', otherwise tomorrow's date. Returns (zero time, false) on no
// match. Caller passes 'now' so tests are deterministic.
//
// window should be the last ~2KB of PTY output for the session; the
// banner is at most ~80 bytes so a 2KB buffer never truncates a fresh
// occurrence.
func ScanForRateLimitBanner(window []byte, now time.Time) (time.Time, bool) {
	m := rateLimitBannerRe.FindSubmatch(window)
	if len(m) < 2 {
		return time.Time{}, false
	}
	reset, err := parseResetTime(string(m[1]), now)
	if err != nil {
		return time.Time{}, false
	}
	return reset, true
}

// parseResetTime turns a fragment like "10:20am" or "22:30" into a
// concrete UTC time anchored relative to 'now'. If the parsed
// time-of-day is in the past for today, we roll to tomorrow — Claude
// banners are forward-looking ("resets at HH:MM") so a past time
// implies tomorrow.
func parseResetTime(s string, now time.Time) (time.Time, error) {
	// Capture group may include trailing space the greedy \s* picked
	// up before failing the am/pm alternative. time.Parse is strict
	// about extra characters, so normalize first.
	s = strings.TrimSpace(s)
	// Try a series of acceptable layouts. Times in claude banners
	// vary between 24h (no am/pm) and 12h (with am/pm); we accept
	// both.
	layouts := []string{
		"3:04pm", "3:04 pm", "3pm", "3 pm",
		"3:04PM", "3:04 PM", "3PM", "3 PM",
		"15:04", "15",
	}
	var parsed time.Time
	var err error
	for _, l := range layouts {
		parsed, err = time.Parse(l, s)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, err
	}
	// Anchor to today in UTC.
	now = now.UTC()
	reset := time.Date(
		now.Year(), now.Month(), now.Day(),
		parsed.Hour(), parsed.Minute(), 0, 0, time.UTC,
	)
	if !reset.After(now) {
		reset = reset.Add(24 * time.Hour)
	}
	return reset, nil
}
