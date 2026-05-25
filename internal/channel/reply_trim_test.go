package channel

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReplyMaxChars_Parsing(t *testing.T) {
	cases := []struct {
		raw  string // raw JSON value for reply_max_chars, or "" for absent
		want int
	}{
		{"", defaultReplyMaxChars},      // absent → default
		{`"3500"`, 3500},                // dashboard submits a string
		{`"0"`, 0},                      // explicit unlimited as string
		{`5000`, 5000},                  // hand-edited as a number
		{`0`, 0},                        // unlimited as number
		{`""`, defaultReplyMaxChars},    // empty string → default
		{`"  900 "`, 900},               // whitespace tolerated
		{`"abc"`, defaultReplyMaxChars}, // garbage → default
		{`-1`, defaultReplyMaxChars},    // negative → default
		{`"-5"`, defaultReplyMaxChars},  // negative string → default
	}
	for _, c := range cases {
		var cc chatConfig
		if c.raw != "" {
			cc.ReplyMaxChars = json.RawMessage(c.raw)
		}
		if got := cc.replyMaxChars(); got != c.want {
			t.Errorf("replyMaxChars(%q) = %d, want %d", c.raw, got, c.want)
		}
	}
}

func TestTrimReply(t *testing.T) {
	t.Run("under cap is untouched", func(t *testing.T) {
		body, footer := trimReply("short reply", 100)
		if body != "short reply" || footer != "" {
			t.Errorf("got (%q,%q), want (%q,%q)", body, footer, "short reply", "")
		}
	})

	t.Run("unlimited keeps whole reply", func(t *testing.T) {
		long := strings.Repeat("x", 10000)
		body, footer := trimReply(long, 0)
		if body != long || footer != "" {
			t.Errorf("max=0 must not trim: bodyLen=%d footer=%q", len(body), footer)
		}
	})

	t.Run("over cap trims with footer", func(t *testing.T) {
		long := strings.Repeat("a", 5000)
		body, footer := trimReply(long, 3500)
		if len([]rune(body)) > 3500 {
			t.Errorf("body exceeds cap: %d", len([]rune(body)))
		}
		if !strings.Contains(footer, "truncated") {
			t.Errorf("footer missing truncation note: %q", footer)
		}
		if !strings.Contains(footer, "1500") {
			t.Errorf("footer should report 1500 dropped chars, got %q", footer)
		}
	})

	t.Run("prefers a line boundary", func(t *testing.T) {
		// 40 lines of "line N"; cap mid-way should cut at a newline so
		// the body ends cleanly rather than mid-line.
		var b strings.Builder
		for i := 0; i < 40; i++ {
			b.WriteString("line of text here\n")
		}
		body, footer := trimReply(b.String(), 100)
		if footer == "" {
			t.Fatal("expected trimming")
		}
		if strings.HasSuffix(body, "line of text her") {
			t.Errorf("body was cut mid-line: %q", body)
		}
	})
}
