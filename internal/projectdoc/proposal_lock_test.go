package projectdoc

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestApproveProposalPreservesOperatorLock is the Bug 1 regression: approving a
// proposal is an operator decision, so the resulting doc must stay
// operator-authored (HumanLocked). Stamping 'agent' would silently un-lock the
// page, turning the human-lock into a one-shot any approval throws away.
//
// DB-gated (skips without OPENDRAY_DEV_DB_URL). Uses an isolated throwaway cwd
// and cleans up after itself so it never touches real project or global data.
func TestApproveProposalPreservesOperatorLock(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	svc := NewService(pool, nil)
	svc.DisableMirror()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uniq := time.Now().UnixNano()
	cwd := fmt.Sprintf("/__opendray_test__/approve-lock-%d", uniq)
	kind := string(KindGoal)
	docID := fmt.Sprintf("pd_test_%d", uniq)
	propID := fmt.Sprintf("pdp_test_%d", uniq)

	// Registered after `defer pool.Close()` so LIFO runs the deletes FIRST,
	// while the pool is still open (t.Cleanup would run after pool.Close).
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM project_doc_proposals WHERE cwd=$1`, cwd)
		_, _ = pool.Exec(context.Background(), `DELETE FROM project_docs WHERE cwd=$1`, cwd)
	}()

	// Seed an operator-locked doc + a pending proposal directly. Raw INSERTs
	// bypass blueprint validation, which is irrelevant to what we're testing.
	if _, err := pool.Exec(ctx,
		`INSERT INTO project_docs (id,cwd,kind,content,updated_by) VALUES ($1,$2,$3,$4,'operator')`,
		docID, cwd, kind, "locked body"); err != nil {
		t.Fatalf("seed doc: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO project_doc_proposals (id,cwd,kind,proposed_content,reason) VALUES ($1,$2,$3,$4,$5)`,
		propID, cwd, kind, "refreshed body", "test"); err != nil {
		t.Fatalf("seed proposal: %v", err)
	}

	doc, err := svc.ApproveProposal(ctx, propID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if doc.UpdatedBy != AuthorOperator {
		t.Fatalf("approved doc author = %q, want %q (approving must preserve the operator lock)",
			doc.UpdatedBy, AuthorOperator)
	}
	if doc.Content != "refreshed body" {
		t.Fatalf("approved content = %q, want %q", doc.Content, "refreshed body")
	}
}

// TestRejectedProposalContents verifies the query the drafter uses to suppress
// re-proposing rejected refreshes (Bug 2) returns only rejected rows.
func TestRejectedProposalContents(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	svc := NewService(pool, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uniq := time.Now().UnixNano()
	cwd := fmt.Sprintf("/__opendray_test__/rejected-%d", uniq)
	kind := string(KindInfrastructure)

	// defer (not t.Cleanup) so the delete runs before `defer pool.Close()`.
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM project_doc_proposals WHERE cwd=$1`, cwd)
	}()

	seedDecided := func(id, content, decision string) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO project_doc_proposals (id,cwd,kind,proposed_content,reason,decision,decided_at) VALUES ($1,$2,$3,$4,'r',$5,NOW())`,
			id, cwd, kind, content, decision); err != nil {
			t.Fatalf("seed %s: %v", decision, err)
		}
	}
	seedPending := func(id, content string) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO project_doc_proposals (id,cwd,kind,proposed_content,reason) VALUES ($1,$2,$3,$4,'r')`,
			id, cwd, kind, content); err != nil {
			t.Fatalf("seed pending: %v", err)
		}
	}
	seedDecided(fmt.Sprintf("pdp_r_%d", uniq), "REJECTED body", "rejected")
	seedPending(fmt.Sprintf("pdp_p_%d", uniq+1), "PENDING body")
	seedDecided(fmt.Sprintf("pdp_a_%d", uniq+2), "APPROVED body", "approved")

	got, err := svc.RejectedProposalContents(ctx, cwd, KindInfrastructure)
	if err != nil {
		t.Fatalf("rejected contents: %v", err)
	}
	if len(got) != 1 || got[0] != "REJECTED body" {
		t.Fatalf("got %v, want [REJECTED body]", got)
	}
}
