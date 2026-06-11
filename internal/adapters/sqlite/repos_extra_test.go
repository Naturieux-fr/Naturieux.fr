package sqlite_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Naturieux-fr/Naturieux.fr/internal/adapters/sqlite"
	"github.com/Naturieux-fr/Naturieux.fr/internal/domain/gamification"
)

func memDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func TestInviteRepository(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewInviteRepository(db)
	ctx := context.Background()

	if err := r.CreateInvite(ctx, "tok", "admin", now()); err != nil {
		t.Fatalf("create: %v", err)
	}
	inv, err := r.GetInvite(ctx, "tok")
	if err != nil || inv.Token != "tok" || inv.CreatedBy != "admin" {
		t.Fatalf("get = %+v, %v", inv, err)
	}
	min := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	ok, err := r.ConsumeInvite(ctx, "tok", "bob", now(), min)
	if err != nil || !ok {
		t.Fatalf("consume = %v,%v", ok, err)
	}
	// Reused -> not consumed again.
	if ok, _ := r.ConsumeInvite(ctx, "tok", "x", now(), min); ok {
		t.Error("invite should be single-use")
	}
	list, _ := r.ListInvites(ctx, 10)
	if len(list) != 1 {
		t.Errorf("list len = %d, want 1", len(list))
	}
	// Revoke an unknown token -> not found.
	if err := r.RevokeInvite(ctx, "nope"); err == nil {
		t.Error("revoking unknown invite should error")
	}
}

func TestChallengeRepository(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewChallengeRepository(db)
	ctx := context.Background()

	_ = r.SaveScore(ctx, "daily", "2026-06-11", "p1", "Alice", 100, 5, 10, now())
	_ = r.SaveScore(ctx, "daily", "2026-06-11", "p1", "Alice", 80, 4, 10, now()) // lower, keep best
	_ = r.SaveScore(ctx, "daily", "2026-06-11", "p2", "Bob", 150, 7, 10, now())

	board, err := r.Leaderboard(ctx, "daily", "2026-06-11", 10)
	if err != nil || len(board) != 2 {
		t.Fatalf("leaderboard = %d, %v", len(board), err)
	}
	if board[0].Username != "Bob" || board[0].Rank != 1 {
		t.Errorf("top = %+v, want Bob #1", board[0])
	}
	best, played, _ := r.BestFor(ctx, "daily", "2026-06-11", "p1")
	if best != 100 || !played {
		t.Errorf("best p1 = %d,%v, want 100,true", best, played)
	}
	if _, played, _ := r.BestFor(ctx, "daily", "2026-06-11", "ghost"); played {
		t.Error("ghost should not have played")
	}
}

func TestCuratedRepository(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewCuratedRepository(db)
	ctx := context.Background()

	if err := r.CreateQuiz(ctx, "q1", "Rapaces", []int{1, 2, 3}, now()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if list, _ := r.ListQuizzes(ctx); len(list) != 1 || list[0].Name != "Rapaces" {
		t.Fatalf("list = %+v", list)
	}
	if sp, _ := r.QuizSpecies(ctx, "q1"); len(sp) != 3 {
		t.Errorf("species = %v", sp)
	}
	_ = r.Schedule(ctx, "daily", "2026-06-11", "q1")
	if id, ok, _ := r.Scheduled(ctx, "daily", "2026-06-11"); !ok || id != "q1" {
		t.Errorf("scheduled = %q,%v", id, ok)
	}
	if err := r.DeleteQuiz(ctx, "q1"); err != nil {
		t.Errorf("delete: %v", err)
	}
	if _, ok, _ := r.Scheduled(ctx, "daily", "2026-06-11"); ok {
		t.Error("schedule should be gone after quiz delete")
	}
}

func TestArticleRepository(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewArticleRepository(db)
	ctx := context.Background()

	a := sqlite.Article{
		ID: "a1", Title: "Triton", Body: "…", Published: true,
		Species:   []sqlite.ArticleSpecies{{CdNom: 92, Name: "Sp"}},
		CreatedAt: now(), UpdatedAt: now(),
	}
	if err := r.Save(ctx, a); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := r.Get(ctx, "a1")
	if err != nil || got.Title != "Triton" || len(got.Species) != 1 {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if pub, _ := r.List(ctx, true); len(pub) != 1 {
		t.Errorf("published list = %d", len(pub))
	}
	if by, _ := r.BySpecies(ctx, 92); len(by) != 1 {
		t.Errorf("by-species = %d", len(by))
	}
	if by, _ := r.BySpecies(ctx, 999); len(by) != 0 {
		t.Errorf("by-species unknown = %d", len(by))
	}
	if err := r.Delete(ctx, "a1"); err != nil {
		t.Errorf("delete: %v", err)
	}
	if _, err := r.Get(ctx, "a1"); err == nil {
		t.Error("get after delete should fail")
	}
}

func TestMissRepository(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewMissRepository(db)
	ctx := context.Background()

	_ = r.RecordMiss(ctx, "p1", 10)
	_ = r.RecordMiss(ctx, "p1", 10)
	_ = r.RecordMiss(ctx, "p1", 20)
	top, _ := r.TopMisses(ctx, "p1", 10)
	if len(top) != 2 || top[0] != 10 { // 10 missed twice -> first
		t.Errorf("top misses = %v, want [10 20]", top)
	}
	// Decay 20 to zero -> leaves the pool.
	_ = r.DecayMiss(ctx, "p1", 20)
	top, _ = r.TopMisses(ctx, "p1", 10)
	if len(top) != 1 || top[0] != 10 {
		t.Errorf("after decay = %v, want [10]", top)
	}
}

func TestResetRepository(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewResetRepository(db)
	ctx := context.Background()

	_ = r.CreateReset(ctx, "rtok", "p1", now())
	min := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	pid, ok, err := r.ConsumeReset(ctx, "rtok", min)
	if err != nil || !ok || pid != "p1" {
		t.Fatalf("consume = %q,%v,%v", pid, ok, err)
	}
	if _, ok, _ := r.ConsumeReset(ctx, "rtok", min); ok {
		t.Error("reset should be single-use")
	}
	// Expired (created in the future of minCreatedAt window) -> rejected.
	_ = r.CreateReset(ctx, "old", "p2", "2000-01-01T00:00:00Z")
	if _, ok, _ := r.ConsumeReset(ctx, "old", min); ok {
		t.Error("expired reset should be rejected")
	}
}

func TestPlayerAdminStore(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewPlayerRepository(db)
	ctx := context.Background()

	for _, name := range []string{"Alice", "Bob"} {
		p, _ := gamification.NewPlayer("id-"+name, name)
		if err := r.Create(ctx, p); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	if n, _ := r.CountPlayers(ctx); n != 2 {
		t.Errorf("CountPlayers = %d", n)
	}
	if err := r.SetRole(ctx, "id-Alice", "admin"); err != nil {
		t.Fatalf("set role: %v", err)
	}
	if n, _ := r.CountAdmins(ctx); n != 1 {
		t.Errorf("CountAdmins = %d", n)
	}
	if role, _ := r.Role(ctx, "id-Alice"); role != "admin" {
		t.Errorf("Role = %q", role)
	}
	if err := r.SetRole(ctx, "id-Bob", "bogus"); err == nil {
		t.Error("invalid role should be rejected")
	}
	if list, _ := r.ListPlayers(ctx, 10); len(list) != 2 {
		t.Errorf("ListPlayers = %d", len(list))
	}
	if err := r.DeletePlayer(ctx, "id-Bob"); err != nil {
		t.Errorf("delete: %v", err)
	}
	if n, _ := r.CountPlayers(ctx); n != 1 {
		t.Errorf("CountPlayers after delete = %d", n)
	}
}

func TestAccountStoreAndOptimize(t *testing.T) {
	db := memDB(t)
	r := sqlite.NewPlayerRepository(db)
	ctx := context.Background()

	if err := r.UpsertAdmin(ctx, "a1", "boss", "hash", now()); err != nil {
		t.Fatalf("UpsertAdmin: %v", err)
	}
	if err := r.SetCredentials(ctx, "a1", "hash2"); err != nil {
		t.Fatalf("SetCredentials: %v", err)
	}
	acc, err := r.Credentials(ctx, "boss")
	if err != nil || acc.ID != "a1" || acc.Role != "admin" {
		t.Errorf("Credentials = %+v, %v", acc, err)
	}
	if _, err := r.TotalGames(ctx); err != nil {
		t.Errorf("TotalGames: %v", err)
	}
	if err := sqlite.Optimize(db); err != nil {
		t.Errorf("Optimize: %v", err)
	}
}
