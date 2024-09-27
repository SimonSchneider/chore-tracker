package chore_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/SimonSchneider/go-testing/chore"
	"github.com/SimonSchneider/go-testing/date"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestDurationMarshaling(t *testing.T) {
	c := date.Duration(time.Hour)
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("unexpected error: marshal: %v", err)
	}
	if string(b) != `"1h0m0s"` {
		t.Fatalf("unexpected json: %s", string(b))
	}
	var parsed date.Duration
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unexpected error: unmarshal: %v", err)
	}
	if parsed != c {
		t.Fatalf("different duration after parsing: inital '%v' parsed: '%v'", c, parsed)
	}
}

func TestDB(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	if err := chore.Setup(ctx, db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ch, err := chore.Create(ctx, db, chore.Input{Name: "simon", Interval: Must(date.ParseDuration("24h"))})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	t.Logf("added chore: %+v", ch)
	if err := chore.Complete(ctx, db, ch.ID, date.Today()); err != nil {
		t.Fatalf("couldn't complete: %v", err)
	}
	chores, err := chore.List(ctx, db)
	t.Logf("chores: %+v", chores)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chores) != 1 {
		t.Fatalf("unexpected number of chores: %d", len(chores))
	}
	chore := chores[0]
	if chore.ID != ch.ID {
		t.Fatalf("unexpected id: %s", chore.ID)
	}
}

func MustT(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

func testCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}

func setup(ctx context.Context) (*http.ServeMux, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	if err := chore.Setup(ctx, db); err != nil {
		return nil, fmt.Errorf("setup db: %w", err)
	}
	panic("not implemented")
}

func createBody(name string, interval string) *chore.Input {
	return &chore.Input{
		Name:     name,
		Interval: Must(date.ParseDuration(interval)),
	}
}

func Exec(ctx context.Context, mux *http.ServeMux, method string, path string, body any) *http.Response {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(Must(json.Marshal(body)))
	}
	req := httptest.NewRequestWithContext(ctx, method, path, bodyReader)
	fmt.Printf("sending req: %s %s\n", method, path)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w.Result()
}

func ExecDec(ctx context.Context, mux *http.ServeMux, method string, path string, body any, v any) error {
	res := Exec(ctx, mux, method, path, body)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("req(%s %s): unexpected status(%d): %s", method, path, res.StatusCode, res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		return fmt.Errorf("req(%s %s): decode response body: %w", method, path, err)
	}
	return nil
}

func TestCreateAndComplete(t *testing.T) {
	ctx := testCtx(t)
	mux := Must(setup(ctx))

	var ch chore.Chore
	err := ExecDec(ctx, mux, "POST", "/", createBody("vacuum", "7h"), &ch)
	MustT(t, err, "create chore")
	t.Logf("ch: %+v", ch)
	id := ch.ID
	MustT(t, ExecDec(ctx, mux, "POST", "/"+id+"/completion/", nil, &ch), "completing the chore")
	if len(ch.History) != 1 {
		t.Fatalf("unexpected history length after 1 completion: %s", err)
	}
	MustT(t, ExecDec(ctx, mux, "POST", "/"+id+"/completion/", nil, &ch), "completing a second time")
	if len(ch.History) != 2 {
		t.Fatalf("unexpected history length after 2 completions: %s", err)
	}
	t.Logf("ch: %+v", ch)
	if res := Exec(ctx, mux, "DELETE", "/"+id+"/completion/"+ch.History[0].ID, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected status code: %d", res.StatusCode)
	}
	if len(ch.History) != 1 {
		t.Fatalf("unexpected history length after 1 deletion: %s", err)
	}
}
