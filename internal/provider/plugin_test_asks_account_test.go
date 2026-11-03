package provider

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/testenv"
)

// A provider's model test asks an account that serves the model, as a
// request would (#4992): a newapi plugin with three accounts, one the
// user kept for a cheap model alone and another the only one with
// grok-4.7, tested grok-4.7 with the first account and failed, while the
// gateway routed real requests to the account that serves it. The test
// path takes the same choice, and the model's answer is the one the
// account that serves it gives: which account a test asked is read from
// the vendor there for the account asked (the fake plugin speaks the
// signed-in account's access token as the bearer, so the one asked is
// read from the Authorization header).
func TestModelTestAsksTheAccountThatServesIt(t *testing.T) {
	bun := testenv.Bun(t)
	claudeHome(t)
	t.Setenv("MAGPIE_BUN", bun)
	t.Setenv("FAKE_GROK", "1")
	t.Cleanup(plugin.Settle)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var mu sync.Mutex
	asked := map[string]int{}
	// the vendor answers grok-4.7 on b@fake alone; a@fake, kept for
	// fake-1, is told the model isn't there, as newapi's tells an
	// account without it
	v := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked[r.Header.Get("Authorization")]++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer b@fake" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":{"message":"This model does not exist","type":"model_not_found"}}`)
			return
		}
		fmt.Fprint(w, `{"id":"x","object":"chat.completion","created":1,"model":"grok-4.7","choices":[{"index":0,"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	})
	testenv.FakeVendor(t, v)
	abs, _ := filepath.Abs("../plugin/testdata/fake/index.js")
	if _, err := plugin.Add(ctx, abs); err != nil {
		t.Fatal(err)
	}
	// two accounts: a@fake serves fake-1 alone, b@fake grok-4.7
	ls := []savedLogin{}
	for _, u := range []string{"a", "b"} {
		k, err := plugin.Import(ctx, "fakeco", map[string]any{"type": "oauth", "refresh": "r-" + u, "access": u + "@fake", "expires": 9e15, "accountId": u + "@fake"})
		if err != nil {
			t.Fatal(err)
		}
		ls = append(ls, savedLogin{Agent: "plugin:fakeco", User: u + "@fake", Home: k, On: true})
	}
	saveLogins(t, ls...)
	if _, err := plugin.Providers(ctx); err != nil {
		t.Fatal(err)
	}
	p, err := Find(PluginID("fakeco"))
	if err != nil {
		t.Fatal(err)
	}
	fp := *p
	fp.AccountModels = map[string][]string{
		"a@fake": {"fake-1"},
		"b@fake": {"grok-4.7"},
	}

	res := fp.TestModels(ctx, []string{"grok-4.7"})
	if len(res) != 1 || !res[0].OK {
		t.Fatalf("grok-4.7's test: %+v", res)
	}
	mu.Lock()
	defer mu.Unlock()
	if n := asked["Bearer b@fake"]; n != 1 {
		t.Errorf("the account that serves grok-4.7 was asked %d times, want 1; asked: %v", n, asked)
	}
	if n := asked["Bearer a@fake"]; n != 0 {
		// a@fake was the first account and the old test path asked it; with
		// the fix, the test goes to the account that serves the model
		t.Errorf("the account kept for fake-1 was asked %d times, want 0", n)
	}
}
