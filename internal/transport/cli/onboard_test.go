package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/storage"
)

func TestOnboardHappy(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 3, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) {
			return []identity.Subscription{{ChannelID: "c1", Title: "AI Agents"}}, nil
		},
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--source", "subscriptions-only", "--interests", "ai-agents,local-llms"}, env, strings.NewReader(""), &out, &errOut, deps)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{"imported 3", "interests set"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout = %q, want %q", out.String(), want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "token")); err != nil {
		t.Fatalf("token file: %v", err)
	}

	db, err := storage.Open(context.Background(), filepath.Join(dir, "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	profile, err := sqlite.NewProfileRepository(db).Get(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Get profile: %v", err)
	}
	if len(profile.Interests) != 2 || profile.Interests[0] != "ai-agents" {
		t.Fatalf("interests = %v", profile.Interests)
	}
	if profile.RubricVersion != "v1" {
		t.Fatalf("rubric = %q", profile.RubricVersion)
	}
}

func TestOnboardSkipInterests(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 1, nil },
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--skip-interests"}, env, strings.NewReader(""), &out, &errOut, deps)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "interests not set") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestOnboardUnavailableSource(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 1, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) {
			return []identity.Subscription{{ChannelID: "c1", Title: "AI Agents"}}, nil
		},
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--source", "external-trends"}, env, strings.NewReader(""), &out, &errOut, deps)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(out.String(), "external-trends") || !strings.Contains(out.String(), "unavailable") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestOnboardImportError(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 0, errors.New("network down") },
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--skip-interests"}, env, strings.NewReader(""), &out, &errOut, deps)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stderr = %s", code, errOut.String())
	}
}

func TestOnboardRevoked(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 0, identity.ErrUnauthorized },
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--skip-interests"}, env, strings.NewReader(""), &out, &errOut, deps)
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr = %s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "re-authenticate") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestOnboardProposalFromChannels(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 2, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) {
			return []identity.Subscription{
				{ChannelID: "c1", Title: "AI Agents", SubscriberCount: 1000},
				{ChannelID: "c2", Title: "Local LLMs", SubscriberCount: 2000},
			}, nil
		},
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--source", "subscriptions-only"}, env, strings.NewReader("\n"), &out, &errOut, deps)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ai-agents") {
		t.Fatalf("stdout = %q", out.String())
	}

	db, err := storage.Open(context.Background(), filepath.Join(dir, "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	profile, err := sqlite.NewProfileRepository(db).Get(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Get profile: %v", err)
	}
	found := false
	for _, interest := range profile.Interests {
		if interest == "ai-agents" {
			found = true
		}
	}
	if !found {
		t.Fatalf("interests = %v", profile.Interests)
	}
}

func TestOnboardExistingTokenSkipsOAuth(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	writeStoredToken(t, dir, identity.Token{AccessToken: "existing", ExpiresAt: time.Now().Add(time.Hour)})
	authorized := false
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			authorized = true
			return identity.Token{}, errors.New("should not authorize")
		},
		Import: func(_ context.Context, token string) (int, error) {
			if token != "existing" {
				t.Fatalf("token = %q", token)
			}
			return 1, nil
		},
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--skip-interests"}, env, strings.NewReader(""), &out, &errOut, deps)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if authorized {
		t.Fatal("onboard ran OAuth with an existing token")
	}
}

func TestOnboardMissingClientFile(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--skip-interests"}, authEnv("", dir), strings.NewReader(""), &out, &errOut, onboardDeps{})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "GOOGLE_OAUTH_CLIENT_FILE") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestOnboardUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--bad"}, authEnv("", t.TempDir()), strings.NewReader(""), &out, &errOut, onboardDeps{})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestOnboardHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--help"}, authEnv("", t.TempDir()), strings.NewReader(""), &out, &errOut, onboardDeps{})
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Usage: ytfeed onboard") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestOnboardInvalidInterests(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 1, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) {
			return []identity.Subscription{{ChannelID: "c1", Title: "AI Agents"}}, nil
		},
	}

	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), []string{"--source", "subscriptions-only", "--interests", "a,b,c,d,e,f,g,h,i,j,k"}, env, strings.NewReader(""), &out, &errOut, deps)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestProvisionalProfileValid(t *testing.T) {
	profile := provisionalProfile("owner", []string{"ai-agents"})
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if profile.RubricVersion != "v1" {
		t.Fatalf("rubric = %q", profile.RubricVersion)
	}
	if len(profile.Axes) < 3 {
		t.Fatalf("axes = %d", len(profile.Axes))
	}
}

func TestOnboardExistingProfileUpdatesInterests(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 1, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) {
			return []identity.Subscription{{ChannelID: "c1", Title: "AI Agents"}}, nil
		},
	}
	for _, interests := range []string{"ai-agents", "ai-agents,local-llms,go"} {
		var out, errOut bytes.Buffer
		if code := runOnboardWith(context.Background(), []string{"--source", "subscriptions-only", "--interests", interests}, env, strings.NewReader(""), &out, &errOut, deps); code != 0 {
			t.Fatalf("run %q exit = %d, stderr = %s", interests, code, errOut.String())
		}
	}

	db, err := storage.Open(context.Background(), filepath.Join(dir, "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	profile, err := sqlite.NewProfileRepository(db).Get(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Get profile: %v", err)
	}
	if len(profile.Interests) != 3 {
		t.Fatalf("interests = %v", profile.Interests)
	}
}

func TestOnboardChannelsError(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 1, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) {
			return nil, errors.New("list failed")
		},
	}
	var out, errOut bytes.Buffer
	if code := runOnboardWith(context.Background(), []string{"--source", "subscriptions-only"}, env, strings.NewReader(""), &out, &errOut, deps); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestOnboardSourcePrompt(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(context.Context, string) (int, error) { return 1, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) {
			return []identity.Subscription{{ChannelID: "c1", Title: "AI Agents"}}, nil
		},
	}
	var out, errOut bytes.Buffer
	if code := runOnboardWith(context.Background(), nil, env, strings.NewReader("subscriptions-only\n"), &out, &errOut, deps); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ai-agents") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestOnboardNoAvailableSource(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import:   func(context.Context, string) (int, error) { return 0, nil },
		Channels: func(context.Context, string) ([]identity.Subscription, error) { return nil, nil },
	}
	var out, errOut bytes.Buffer
	if code := runOnboardWith(context.Background(), nil, env, strings.NewReader(""), &out, &errOut, deps); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestOnboardExpiredTokenReauth(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	writeStoredToken(t, dir, identity.Token{AccessToken: "expired", ExpiresAt: time.Now().Add(-time.Hour)})
	authorized := false
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			authorized = true
			return identity.Token{AccessToken: "new", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		Import: func(_ context.Context, token string) (int, error) {
			if token != "new" {
				t.Fatalf("token = %q", token)
			}
			return 1, nil
		},
	}
	var out, errOut bytes.Buffer
	if code := runOnboardWith(context.Background(), []string{"--skip-interests"}, env, strings.NewReader(""), &out, &errOut, deps); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !authorized {
		t.Fatal("expired token did not trigger OAuth")
	}
}

func TestOnboardTokenReadError(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	if err := os.Mkdir(filepath.Join(dir, "token"), 0o700); err != nil {
		t.Fatalf("mkdir token: %v", err)
	}
	var out, errOut bytes.Buffer
	if code := runOnboardWith(context.Background(), []string{"--skip-interests"}, env, strings.NewReader(""), &out, &errOut, onboardDeps{}); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestOnboardConfigError(t *testing.T) {
	var out, errOut bytes.Buffer
	code := runOnboardWith(context.Background(), nil, func(key string) string {
		if key == "LOG_ENV" {
			return "staging"
		}
		return ""
	}, strings.NewReader(""), &out, &errOut, onboardDeps{})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestProposeFromChannelsFallback(t *testing.T) {
	got := proposeFromChannels(nil)
	if len(got) != 5 {
		t.Fatalf("proposal = %v", got)
	}
}

func TestSlugify(t *testing.T) {
	for input, want := range map[string]string{
		"AI Agents":       "ai-agents",
		"  Local  LLMs  ": "local-llms",
		"---":             "",
		"C++ / Rust":      "c-rust",
	} {
		if got := slugify(input); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" ai-agents, , local-llms ")
	if len(got) != 2 || got[0] != "ai-agents" || got[1] != "local-llms" {
		t.Fatalf("splitCSV = %v", got)
	}
}

func TestOnboardOAuthTimeout(t *testing.T) {
	dir := t.TempDir()
	env := authEnv(writeAuthClient(t, dir), dir)
	deps := onboardDeps{
		Authorize: func(context.Context, identity.Service) (identity.Token, error) {
			return identity.Token{}, identity.ErrTimeout
		},
	}
	var out, errOut bytes.Buffer
	if code := runOnboardWith(context.Background(), []string{"--skip-interests"}, env, strings.NewReader(""), &out, &errOut, deps); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "ytfeed onboard: authorization timed out") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}
