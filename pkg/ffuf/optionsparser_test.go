package ffuf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplatePresent(t *testing.T) {
	template := "§"

	headers := make(map[string]string)
	headers["foo"] = "§bar§"
	headers["omg"] = "bbq"
	headers["§world§"] = "Ooo"

	goodConf := Config{
		Url:     "https://example.com/fooo/bar?test=§value§&order[§0§]=§foo§",
		Method:  "PO§ST§",
		Headers: headers,
		Data:    "line=Can we pull back the §veil§ of §static§ and reach in to the source of §all§ being?&commit=true",
	}

	if !templatePresent(template, &goodConf) {
		t.Errorf("Expected-good config failed validation")
	}

	badConfMethod := Config{
		Url:     "https://example.com/fooo/bar?test=§value§&order[§0§]=§foo§",
		Method:  "POST§",
		Headers: headers,
		Data:    "line=Can we pull back the §veil§ of §static§ and reach in to the source of §all§ being?&commit=§true§",
	}

	if templatePresent(template, &badConfMethod) {
		t.Errorf("Expected-bad config (Method) failed validation")
	}

	badConfURL := Config{
		Url:     "https://example.com/fooo/bar?test=§value§&order[0§]=§foo§",
		Method:  "§POST§",
		Headers: headers,
		Data:    "line=Can we pull back the §veil§ of §static§ and reach in to the source of §all§ being?&commit=§true§",
	}

	if templatePresent(template, &badConfURL) {
		t.Errorf("Expected-bad config (URL) failed validation")
	}

	badConfData := Config{
		Url:     "https://example.com/fooo/bar?test=§value§&order[§0§]=§foo§",
		Method:  "§POST§",
		Headers: headers,
		Data:    "line=Can we pull back the §veil of §static§ and reach in to the source of §all§ being?&commit=§true§",
	}

	if templatePresent(template, &badConfData) {
		t.Errorf("Expected-bad config (Data) failed validation")
	}

	headers["kingdom"] = "§candy"

	badConfHeaderValue := Config{
		Url:     "https://example.com/fooo/bar?test=§value§&order[§0§]=§foo§",
		Method:  "PO§ST§",
		Headers: headers,
		Data:    "line=Can we pull back the §veil§ of §static§ and reach in to the source of §all§ being?&commit=true",
	}

	if templatePresent(template, &badConfHeaderValue) {
		t.Errorf("Expected-bad config (Header value) failed validation")
	}

	headers["kingdom"] = "candy"
	headers["§kingdom"] = "candy"

	badConfHeaderKey := Config{
		Url:     "https://example.com/fooo/bar?test=§value§&order[§0§]=§foo§",
		Method:  "PO§ST§",
		Headers: headers,
		Data:    "line=Can we pull back the §veil§ of §static§ and reach in to the source of §all§ being?&commit=true",
	}

	if templatePresent(template, &badConfHeaderKey) {
		t.Errorf("Expected-bad config (Header key) failed validation")
	}
}

func TestProxyParsing(t *testing.T) {
	configOptions := NewConfigOptions()
	errorString := "Bad proxy url (-x) format. Expected http, https or socks5 url"

	// http should work
	configOptions.HTTP.ProxyURL = "http://127.0.0.1:8080"
	_, err := ConfigFromOptions(configOptions, nil, nil)
	if strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected http proxy string to work")
	}

	// https should work
	configOptions.HTTP.ProxyURL = "https://127.0.0.1"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected https proxy string to work")
	}

	// socks5 should work
	configOptions.HTTP.ProxyURL = "socks5://127.0.0.1"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected socks5 proxy string to work")
	}

	// garbage data should FAIL
	configOptions.HTTP.ProxyURL = "Y0 y0 it's GREASE"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if !strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected garbage proxy string to fail")
	}

	// Opaque URLs with the right scheme should FAIL
	configOptions.HTTP.ProxyURL = "http:sixhours@dungeon"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if !strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected opaque proxy string to fail")
	}

	// Unsupported protocols should FAIL
	configOptions.HTTP.ProxyURL = "imap://127.0.0.1"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if !strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected proxy string with unsupported protocol to fail")
	}
}

func TestReplayProxyParsing(t *testing.T) {
	configOptions := NewConfigOptions()
	errorString := "Bad replay-proxy url (-replay-proxy) format. Expected http, https or socks5 url"

	// http should work
	configOptions.HTTP.ReplayProxyURL = "http://127.0.0.1:8080"
	_, err := ConfigFromOptions(configOptions, nil, nil)
	if strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected http replay proxy string to work")
	}

	// https should work
	configOptions.HTTP.ReplayProxyURL = "https://127.0.0.1"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected https proxy string to work")
	}

	// socks5 should work
	configOptions.HTTP.ReplayProxyURL = "socks5://127.0.0.1"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected socks5 proxy string to work")
	}

	// garbage data should FAIL
	configOptions.HTTP.ReplayProxyURL = "Y0 y0 it's GREASE"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if !strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected garbage proxy string to fail")
	}

	// Opaque URLs with the right scheme should FAIL
	configOptions.HTTP.ReplayProxyURL = "http:sixhours@dungeon"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if !strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected opaque proxy string to fail")
	}

	// Unsupported protocols should FAIL
	configOptions.HTTP.ReplayProxyURL = "imap://127.0.0.1"
	_, err = ConfigFromOptions(configOptions, nil, nil)
	if !strings.Contains(err.Error(), errorString) {
		t.Errorf("Expected proxy string with unsupported protocol to fail")
	}
}

func writeParserTestWordlist(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Two wordlists without explicit keywords both default to FUZZ. The input
// value model keys values by keyword, so this used to silently collapse: the
// first wordlist's payloads were never sent while the reported combination
// count still covered both lists. The configuration must now be rejected.
func TestConfigFromOptionsRejectsDuplicateDefaultKeyword(t *testing.T) {
	w1 := writeParserTestWordlist(t, "w1.txt")
	w2 := writeParserTestWordlist(t, "w2.txt")

	opts := NewConfigOptions()
	opts.HTTP.URL = "https://example.org/FUZZ"
	opts.Input.Wordlists = []string{w1, w2}

	_, err := ConfigFromOptions(opts, context.Background(), func() {})
	if err == nil || !strings.Contains(err.Error(), "multiple input providers") {
		t.Fatalf("expected duplicate keyword error, got: %v", err)
	}
}

// A wordlist and an input command sharing the (default) keyword collapse the
// same way and must be rejected as well.
func TestConfigFromOptionsRejectsDuplicateKeywordAcrossWordlistAndCommand(t *testing.T) {
	w1 := writeParserTestWordlist(t, "w1.txt")

	opts := NewConfigOptions()
	opts.HTTP.URL = "https://example.org/FUZZ"
	opts.Input.Wordlists = []string{w1}
	opts.Input.Inputcommands = []string{"printf fuzz"}

	_, err := ConfigFromOptions(opts, context.Background(), func() {})
	if err == nil || !strings.Contains(err.Error(), "multiple input providers") {
		t.Fatalf("expected duplicate keyword error, got: %v", err)
	}
}

// A trailing colon ("wordlist.txt:") yields an empty keyword, which used to be
// silently assembled. It must be rejected for wordlists and input commands.
func TestConfigFromOptionsRejectsEmptyKeyword(t *testing.T) {
	w1 := writeParserTestWordlist(t, "w1.txt")

	opts := NewConfigOptions()
	opts.HTTP.URL = "https://example.org/FUZZ"
	opts.Input.Wordlists = []string{w1 + ":"}

	_, err := ConfigFromOptions(opts, context.Background(), func() {})
	if err == nil || !strings.Contains(err.Error(), "empty keyword") {
		t.Fatalf("expected empty keyword error for wordlist, got: %v", err)
	}

	opts = NewConfigOptions()
	opts.HTTP.URL = "https://example.org/FUZZ"
	opts.Input.Inputcommands = []string{"printf fuzz:"}

	_, err = ConfigFromOptions(opts, context.Background(), func() {})
	if err == nil || !strings.Contains(err.Error(), "empty keyword") {
		t.Fatalf("expected empty keyword error for input command, got: %v", err)
	}
}

// Control group: explicitly distinct keywords keep assembling both providers.
func TestConfigFromOptionsDistinctKeywordsControl(t *testing.T) {
	w1 := writeParserTestWordlist(t, "w1.txt")
	w2 := writeParserTestWordlist(t, "w2.txt")

	opts := NewConfigOptions()
	opts.HTTP.URL = "https://example.org/W1/W2"
	opts.Input.Wordlists = []string{w1 + ":W1", w2 + ":W2"}

	conf, err := ConfigFromOptions(opts, context.Background(), func() {})
	if err != nil {
		t.Fatalf("distinct keywords must keep working, got: %v", err)
	}
	if len(conf.InputProviders) != 2 {
		t.Fatalf("got %d input providers, want 2", len(conf.InputProviders))
	}
	if conf.InputProviders[0].Keyword != "W1" || conf.InputProviders[1].Keyword != "W2" {
		t.Fatalf("keywords = %q, %q; want W1, W2",
			conf.InputProviders[0].Keyword, conf.InputProviders[1].Keyword)
	}
}
