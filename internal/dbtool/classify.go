package dbtool

import (
	"errors"
	"fmt"
	"strings"
)

// Statement classification for the read-only gate.
//
// This is the FIRST of two fences: it routes a console statement to the
// read gate (db:read, read-only tx) or the write gate (db:write, rejected
// on read_only connections). It is intentionally conservative — anything
// it can't prove is a read is classified as a write, and the read path is
// additionally executed inside a server-side read-only transaction, so a
// misclassification can block a legitimate read but never lets a write
// slip through the read gate.
//
// v1 accepts a single statement only; transaction control is rejected
// outright because every execution already runs inside a managed
// transaction.

// ErrMultiStatement is returned for `a; b` inputs.
var ErrMultiStatement = errors.New("dbtool: multiple statements are not supported; run one statement at a time")

// ErrEmptyStatement is returned when the input is blank / comments only.
var ErrEmptyStatement = errors.New("dbtool: empty statement")

// ErrTxnControl is returned for BEGIN/COMMIT/ROLLBACK etc.
var ErrTxnControl = errors.New("dbtool: transaction control statements are not allowed (every statement already runs in a managed transaction)")

// readLeaders are first keywords that make a statement a read on their own.
var readLeaders = map[string]bool{
	"SELECT": true,
	"VALUES": true,
	"TABLE":  true,
	"SHOW":   true,
}

// txnKeywords are rejected outright.
var txnKeywords = map[string]bool{
	"BEGIN": true, "START": true, "COMMIT": true, "END": true,
	"ROLLBACK": true, "SAVEPOINT": true, "RELEASE": true, "ABORT": true,
}

// ddlLeaders are first keywords classified as DDL (gated like writes).
var ddlLeaders = map[string]bool{
	"CREATE": true, "ALTER": true, "DROP": true, "TRUNCATE": true,
	"GRANT": true, "REVOKE": true, "COMMENT": true, "REINDEX": true,
	"CLUSTER": true, "VACUUM": true, "REFRESH": true, "SECURITY": true,
}

// writeKeywords, appearing as a token ANYWHERE in a WITH statement, make
// the whole statement a write (data-modifying CTEs).
var writeKeywords = map[string]bool{
	"INSERT": true, "UPDATE": true, "DELETE": true, "MERGE": true, "COPY": true,
}

// Classify tokenizes sql (comments and string literals stripped) and
// returns its class, or an error for empty / multi-statement /
// transaction-control input.
func Classify(sql string) (StatementClass, error) {
	tokens := tokenize(sql)
	if len(tokens) == 0 {
		return "", ErrEmptyStatement
	}
	// Multi-statement: a top-level ';' followed by anything else.
	for i, tok := range tokens {
		if tok == ";" && i < len(tokens)-1 {
			return "", ErrMultiStatement
		}
	}
	if tokens[len(tokens)-1] == ";" {
		tokens = tokens[:len(tokens)-1]
	}
	if len(tokens) == 0 {
		return "", ErrEmptyStatement
	}

	first := tokens[0]
	switch {
	case txnKeywords[first]:
		return "", ErrTxnControl
	case readLeaders[first]:
		return ClassRead, nil
	case first == "EXPLAIN":
		return classifyExplain(tokens[1:])
	case first == "WITH":
		// A data-modifying CTE (WITH x AS (INSERT …)) makes the whole
		// statement a write no matter what the final verb is. Scanning
		// every token can false-positive on a column literally named
		// "update" — acceptable: it only routes the read to the stricter
		// gate, and identifiers in quotes are skipped by the tokenizer.
		for _, tok := range tokens[1:] {
			if writeKeywords[tok] {
				return ClassWrite, nil
			}
		}
		return ClassRead, nil
	case writeKeywords[first]:
		return ClassWrite, nil
	case ddlLeaders[first]:
		return ClassDDL, nil
	default:
		// Unknown leader (DO, CALL, SET, LOCK, PREPARE, LISTEN, …):
		// fail safe as a write so it needs db:write and a writable
		// connection.
		return ClassWrite, nil
	}
}

// classifyExplain handles EXPLAIN [( options )] [ANALYZE|VERBOSE] <stmt>.
// Plain EXPLAIN only plans (read); EXPLAIN ANALYZE executes, so it takes
// the class of the inner statement.
func classifyExplain(rest []string) (StatementClass, error) {
	if len(rest) == 0 {
		return "", fmt.Errorf("dbtool: incomplete EXPLAIN statement")
	}
	analyze := false
	i := 0
	// Option list form: EXPLAIN (ANALYZE, BUFFERS) …
	if rest[i] == "(" {
		depth := 0
		for ; i < len(rest); i++ {
			switch rest[i] {
			case "(":
				depth++
			case ")":
				depth--
			case "ANALYZE":
				analyze = true
			}
			if depth == 0 {
				i++
				break
			}
		}
	}
	// Bare keyword form: EXPLAIN ANALYZE VERBOSE …
	for i < len(rest) && (rest[i] == "ANALYZE" || rest[i] == "VERBOSE") {
		if rest[i] == "ANALYZE" {
			analyze = true
		}
		i++
	}
	if !analyze {
		return ClassRead, nil
	}
	if i >= len(rest) {
		return "", fmt.Errorf("dbtool: incomplete EXPLAIN ANALYZE statement")
	}
	inner, err := Classify(strings.Join(rest[i:], " "))
	if err != nil {
		return "", err
	}
	return inner, nil
}

// tokenize splits sql into uppercase word tokens plus "(", ")" and ";"
// punctuation, skipping whitespace, line comments (--), nested block
// comments (slash-star), single-quoted strings (doubled-quote and
// backslash escapes), dollar-quoted strings ($tag$ ... $tag$) and
// double-quoted identifiers (emitted as a placeholder token so quoted
// identifiers never look like keywords).
func tokenize(sql string) []string {
	var tokens []string
	src := []rune(sql)
	n := len(src)
	i := 0
	emitWord := func(start, end int) {
		if end > start {
			tokens = append(tokens, strings.ToUpper(string(src[start:end])))
		}
	}
	for i < n {
		c := src[i]
		switch {
		case c == '-' && i+1 < n && src[i+1] == '-':
			for i < n && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && src[i+1] == '*':
			depth := 1
			i += 2
			for i < n && depth > 0 {
				if src[i] == '/' && i+1 < n && src[i+1] == '*' {
					depth++
					i += 2
				} else if src[i] == '*' && i+1 < n && src[i+1] == '/' {
					depth--
					i += 2
				} else {
					i++
				}
			}
		case c == '\'':
			i++
			for i < n {
				if src[i] == '\'' {
					if i+1 < n && src[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				// Backslash escape inside E'…' strings; harmless to honor
				// for plain strings too (worst case we skip one quote char
				// inside a literal we're discarding anyway).
				if src[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				i++
			}
		case c == '"':
			i++
			for i < n {
				if src[i] == '"' {
					if i+1 < n && src[i+1] == '"' {
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			tokens = append(tokens, `"ident"`)
		case c == '$':
			// Dollar-quoted string: $tag$ … $tag$ (tag may be empty; tag
			// chars are letters/digits/underscore — NOT '$' itself).
			j := i + 1
			for j < n && isIdentRune(src[j]) && src[j] != '$' {
				j++
			}
			if j < n && src[j] == '$' {
				tag := src[i : j+1]
				pos := j + 1
				for ; pos <= n-len(tag); pos++ {
					if string(src[pos:pos+len(tag)]) == string(tag) {
						break
					}
				}
				if pos > n-len(tag) {
					i = n // unterminated — consume the rest
				} else {
					i = pos + len(tag)
				}
			} else {
				// Positional param ($1) or stray '$' — treat as word char.
				start := i
				i++
				for i < n && isIdentRune(src[i]) {
					i++
				}
				emitWord(start, i)
			}
		case c == '(' || c == ')' || c == ';':
			tokens = append(tokens, string(c))
			i++
		case isIdentRune(c):
			start := i
			for i < n && isIdentRune(src[i]) {
				i++
			}
			emitWord(start, i)
		default:
			i++
		}
	}
	return tokens
}

func isIdentRune(c rune) bool {
	return c == '_' || c == '$' ||
		(c >= '0' && c <= '9') ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		c > 127
}
