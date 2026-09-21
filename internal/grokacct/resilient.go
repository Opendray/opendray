package grokacct

import "context"

// homeCandidate is a GROK_HOME the session could spawn under.
type homeCandidate struct {
	name   string
	home   string
	usable bool // enabled AND has a logged-in token on disk
}

// pickSpawnHome keeps a grok session working across whatever accounts are
// available: prefer the pinned account, then the gateway default home,
// then any other logged-in account. Errors only when nothing is usable.
// Pure (no DB / filesystem) so it's unit-testable.
func pickSpawnHome(pinned, dflt homeCandidate, others []homeCandidate) (homeCandidate, error) {
	if pinned.usable {
		return pinned, nil
	}
	if dflt.usable {
		return dflt, nil
	}
	for _, c := range others {
		if c.usable {
			return c, nil
		}
	}
	return homeCandidate{}, ErrNoUsableGrok
}

// CheckUsable returns nil only when id is an existing, enabled account with
// a logged-in token on disk. Used to guard an explicit account switch so a
// switch to a logged-out account is rejected up-front (400) instead of
// stopping the session and then failing to respawn.
func (s *Service) CheckUsable(ctx context.Context, id string) error {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if !a.Enabled {
		return ErrDisabled
	}
	if !accountHasCredentials(a.ConfigDir) {
		return ErrNotLoggedIn
	}
	return nil
}

// ResolveSpawnHomeResilient resolves a GROK_HOME to spawn under, preferring
// account id but falling back to the default home and then any other
// logged-in account, so a session is never bricked by a logged-out
// binding. Returns the home and the account label actually used (which may
// differ from id when a fallback kicks in). id == "" means "prefer the
// gateway default home". Errors only when no logged-in grok home exists.
func (s *Service) ResolveSpawnHomeResilient(ctx context.Context, id string) (home, used string, err error) {
	dfltHome := defaultGrokHome()
	dflt := homeCandidate{name: "default", home: dfltHome, usable: accountHasCredentials(dfltHome)}

	var pinned homeCandidate
	if id != "" {
		if a, e := s.store.Get(ctx, id); e == nil {
			pinned = homeCandidate{name: a.Name, home: a.ConfigDir, usable: a.Enabled && accountHasCredentials(a.ConfigDir)}
		}
	} else {
		// No pin: the default home is the preferred choice.
		pinned = dflt
	}

	var others []homeCandidate
	if accs, e := s.List(ctx); e == nil {
		for _, a := range accs {
			others = append(others, homeCandidate{name: a.Name, home: a.ConfigDir, usable: a.Enabled && a.TokenFilled})
		}
	}

	pick, err := pickSpawnHome(pinned, dflt, others)
	if err != nil {
		return "", "", err
	}
	return pick.home, pick.name, nil
}
