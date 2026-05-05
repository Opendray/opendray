package capture

import "testing"

func TestAfterMessagesTrigger_Evaluate(t *testing.T) {
	cases := []struct {
		name      string
		n         int
		lastSeen  int
		current  int
		wantFire  bool
	}{
		{"first_n_msgs_fires_at_n", 3, -1, 3, true},
		{"first_n_minus_one_does_not_fire", 3, -1, 2, false},
		{"need_n_new_after_last_fire", 5, 4, 9, false}, // current=9 → 9-4-1=4 new, n=5
		{"exact_n_new", 5, 4, 10, true},                // 10-4-1=5 new
		{"more_than_n_new", 5, 4, 100, true},
		{"default_n_when_zero", 0, -1, 6, true},        // n=0 → 6
		{"default_n_when_zero_short", 0, -1, 5, false}, // 5 < 6
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AfterMessagesTrigger{N: c.n}.Evaluate(c.lastSeen, c.current)
			if got != c.wantFire {
				t.Errorf("got fire=%v want %v", got, c.wantFire)
			}
		})
	}
}

func TestTriggerFromRule_AfterMessages(t *testing.T) {
	r := Rule{TriggerKind: "after_messages", TriggerConfig: map[string]any{"n": float64(8)}}
	tg, err := triggerFromRule(r)
	if err != nil {
		t.Fatal(err)
	}
	atm, ok := tg.(AfterMessagesTrigger)
	if !ok {
		t.Fatalf("wrong type %T", tg)
	}
	if atm.N != 8 {
		t.Errorf("N = %d, want 8", atm.N)
	}
}

func TestTriggerFromRule_DefaultsN(t *testing.T) {
	r := Rule{TriggerKind: "after_messages"}
	tg, _ := triggerFromRule(r)
	if !tg.Evaluate(-1, 6) {
		t.Error("default n should be 6")
	}
}

func TestTriggerFromRule_UnsupportedKind(t *testing.T) {
	_, err := triggerFromRule(Rule{TriggerKind: "on_idle"})
	if err == nil {
		t.Error("expected error for unsupported kind")
	}
}
