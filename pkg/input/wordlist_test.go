package input

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ffuf/ffuf/v2/pkg/ffuf"
)

func TestStripCommentsIgnoresCommentLines(t *testing.T) {
	text, _ := stripComments("# text")

	if text != "" {
		t.Errorf("Returned text was not a blank string")
	}
}

func TestStripCommentsStripsCommentAfterText(t *testing.T) {
	text, _ := stripComments("text # comment")

	if text != "text" {
		t.Errorf("Comment was not stripped or pre-comment text was not returned")
	}
}

func TestWordlistAcceptsLinesLargerThanScannerDefault(t *testing.T) {
	// Issue #567 reports a real 448 KiB single-entry wordlist.
	payload := strings.Repeat("a", 448*1024)
	wordlist := filepath.Join(t.TempDir(), "large-wordlist.txt")
	if err := os.WriteFile(wordlist, []byte(payload+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := ffuf.NewConfig(ctx, cancel)
	input, err := NewWordlistInput("FUZZ", wordlist, &config)
	if err != nil {
		t.Fatalf("reading large wordlist entry: %v", err)
	}
	if input.Total() != 1 {
		t.Fatalf("got %d entries, want 1", input.Total())
	}
	if got := string(input.Value()); got != payload {
		t.Fatalf("payload length = %d, want %d", len(got), len(payload))
	}
}

func writeTestWordlist(t *testing.T, lines []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wordlist.txt")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func newTestConfig(t *testing.T, providers ...ffuf.InputProviderConfig) *ffuf.Config {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	config := ffuf.NewConfig(ctx, cancel)
	config.InputMode = "clusterbomb"
	config.InputProviders = providers
	return &config
}

// Two wordlists sharing one keyword (the common case: both defaulted to FUZZ)
// must not silently collapse into a single keyword->value map entry while
// Total() still counts the full cartesian product.
func TestNewInputProviderRejectsDuplicateKeyword(t *testing.T) {
	w1 := writeTestWordlist(t, []string{"alpha", "bravo", "charlie"})
	w2 := writeTestWordlist(t, []string{"delta", "echo"})
	config := newTestConfig(t,
		ffuf.InputProviderConfig{Name: "wordlist", Keyword: "FUZZ", Value: w1},
		ffuf.InputProviderConfig{Name: "wordlist", Keyword: "FUZZ", Value: w2},
	)

	_, errs := NewInputProvider(config)
	if errs.ErrorOrNil() == nil {
		t.Fatal("expected duplicate keyword to be rejected, got no error")
	}
	if !strings.Contains(errs.ErrorOrNil().Error(), "FUZZ") {
		t.Fatalf("error should name the duplicated keyword, got: %v", errs.ErrorOrNil())
	}
}

// An empty keyword (e.g. from a trailing colon in "wordlist.txt:") must not be
// silently assembled either.
func TestNewInputProviderRejectsEmptyKeyword(t *testing.T) {
	w1 := writeTestWordlist(t, []string{"alpha"})
	config := newTestConfig(t,
		ffuf.InputProviderConfig{Name: "wordlist", Keyword: "", Value: w1},
	)

	_, errs := NewInputProvider(config)
	if errs.ErrorOrNil() == nil {
		t.Fatal("expected empty keyword to be rejected, got no error")
	}
	if !strings.Contains(errs.ErrorOrNil().Error(), "empty keyword") {
		t.Fatalf("error should mention the empty keyword, got: %v", errs.ErrorOrNil())
	}
}

// Control group: explicitly distinct keywords keep the full cartesian
// behaviour - every combination is produced exactly once, and every payload of
// the first wordlist appears in the emitted value set.
func TestNewInputProviderDistinctKeywordsCartesian(t *testing.T) {
	w1 := writeTestWordlist(t, []string{"alpha", "bravo", "charlie"})
	w2 := writeTestWordlist(t, []string{"delta", "echo"})
	config := newTestConfig(t,
		ffuf.InputProviderConfig{Name: "wordlist", Keyword: "W1", Value: w1},
		ffuf.InputProviderConfig{Name: "wordlist", Keyword: "W2", Value: w2},
	)

	ip, errs := NewInputProvider(config)
	if errs.ErrorOrNil() != nil {
		t.Fatalf("NewInputProvider: %v", errs.ErrorOrNil())
	}
	if got, want := ip.Total(), 6; got != want {
		t.Fatalf("Total() = %d, want %d", got, want)
	}

	combos := make(map[string]int)
	w1Payloads := make(map[string]bool)
	count := 0
	for ip.Next() {
		v := ip.Value()
		combos[string(v["W1"])+"/"+string(v["W2"])]++
		w1Payloads[string(v["W1"])] = true
		count++
	}
	if count != 6 {
		t.Fatalf("iterated %d combinations, want 6", count)
	}
	if len(combos) != 6 {
		t.Fatalf("got %d distinct combinations, want 6 (each exactly once): %v", len(combos), combos)
	}
	for combo, n := range combos {
		if n != 1 {
			t.Errorf("combination %q emitted %d times, want exactly 1", combo, n)
		}
	}
	if len(w1Payloads) != 3 {
		t.Fatalf("first wordlist contributed %d distinct payloads, want 3: %v", len(w1Payloads), w1Payloads)
	}
}
