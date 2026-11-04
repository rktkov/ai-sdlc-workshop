package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/integration/youtube"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/repository/tokenfile"
	"github.com/rktkov/ytfeed/internal/storage"
)

type onboardDeps struct {
	Authorize func(context.Context, identity.Service) (identity.Token, error)
	Import    func(context.Context, string) (int, error)
	Channels  func(context.Context, string) ([]identity.Subscription, error)
}

type onboardOptions struct {
	source    string
	interests string
	skip      bool
	noBrowser bool
	reset     bool
}

func runOnboard(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runOnboardWith(ctx, args, getenv, stdin, stdout, stderr, onboardDeps{})
}

func runOnboardWith(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer, deps onboardDeps) int {
	opts, code := parseOnboardOptions(args, stdout, stderr)
	if code >= 0 {
		return code
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if deps.Authorize == nil {
		deps.Authorize = func(ctx context.Context, service identity.Service) (identity.Token, error) {
			return service.Authorize(ctx)
		}
	}
	token, code := onboardCredentials(ctx, cfg, *opts, deps, stderr)
	if code != 0 {
		return code
	}

	db, err := storage.Open(ctx, filepath.Join(cfg.Storage.DataDir, "ytfeed.db"))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()

	profiles := sqlite.NewProfileRepository(db)
	subscriptions := sqlite.NewSubscriptionRepository(db)
	tasteService := &taste.Service{Profiles: profiles}
	deps = fillOnboardDeps(deps, subscriptions)
	_, _ = fmt.Fprintln(stdout, "step 1/2: connected Google account")
	count, err := deps.Import(ctx, token.AccessToken)
	if err != nil {
		if errors.Is(err, identity.ErrUnauthorized) {
			_, _ = fmt.Fprintln(stderr, "ytfeed onboard: re-authenticate with `ytfeed auth login`")
			return 2
		}
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: import subscriptions: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "imported %d subscriptions\n", count)

	if opts.skip {
		_, _ = fmt.Fprintln(stdout, "interests not set — feed may be narrow.")
		return 0
	}
	return onboardInterests(ctx, cfg, *opts, bufio.NewReader(stdin), stdout, stderr, deps, profiles, tasteService)
}

func parseOnboardOptions(args []string, stdout, stderr io.Writer) (*onboardOptions, int) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printOnboardHelp(stdout)
			return nil, 0
		}
	}
	flags := flag.NewFlagSet("onboard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	opts := &onboardOptions{}
	flags.StringVar(&opts.source, "source", "", "seed source")
	flags.StringVar(&opts.interests, "interests", "", "comma-separated interests")
	flags.BoolVar(&opts.skip, "skip-interests", false, "skip step 2")
	flags.BoolVar(&opts.noBrowser, "no-browser", false, "print the OAuth URL")
	flags.BoolVar(&opts.reset, "reset", false, "re-run onboarding")
	if err := flags.Parse(args); err != nil {
		return nil, 2
	}
	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: unexpected argument %q\n", flags.Arg(0))
		return nil, 2
	}
	return opts, -1
}

func printOnboardHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: ytfeed onboard [--source NAME] [--interests LIST] [--skip-interests] [--no-browser] [--reset]")
	_, _ = fmt.Fprintln(w, "  --source NAME        seed source: youtube-trending, external-trends, subscriptions-only")
	_, _ = fmt.Fprintln(w, "  --interests LIST     comma-separated interests; skips the proposal prompt")
	_, _ = fmt.Fprintln(w, "  --skip-interests     finish after subscriptions; prints the without-interests banner")
	_, _ = fmt.Fprintln(w, "  --no-browser         print the OAuth URL instead of opening the browser")
	_, _ = fmt.Fprintln(w, "  --reset              re-run both steps; imports are idempotent")
}

func fillOnboardDeps(deps onboardDeps, subscriptions *sqlite.SubscriptionRepository) onboardDeps {
	if deps.Authorize == nil {
		deps.Authorize = func(ctx context.Context, service identity.Service) (identity.Token, error) {
			return service.Authorize(ctx)
		}
	}
	if deps.Import == nil {
		importer := &identity.Importer{
			Source: youtube.NewSource("", nil),
			Store:  subscriptions,
			UserID: "owner",
		}
		deps.Import = importer.Import
	}
	if deps.Channels == nil {
		deps.Channels = subscriptions.List
	}
	return deps
}

func onboardCredentials(ctx context.Context, cfg config.Config, opts onboardOptions, deps onboardDeps, stderr io.Writer) (identity.Token, int) {
	store := tokenfile.New(cfg.Storage.DataDir)
	token, err := store.Read(ctx)
	if err == nil && token.ExpiresAt.After(time.Now().Add(30*time.Second)) {
		return token, 0
	}
	if err != nil && !errors.Is(err, identity.ErrTokenNotFound) {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: read token: %v\n", err)
		return identity.Token{}, 1
	}
	if cfg.YouTube.OAuthClientFile == "" {
		_, _ = fmt.Fprintln(stderr, "ytfeed onboard: GOOGLE_OAUTH_CLIENT_FILE is not set")
		return identity.Token{}, 2
	}
	client, err := identity.LoadClient(cfg.YouTube.OAuthClientFile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: %v\n", err)
		return identity.Token{}, 2
	}

	service := identity.Service{
		Client: client,
		OnAuthURL: func(url string) {
			_, _ = fmt.Fprintf(stderr, "open this URL to authorize ytfeed:\n%s\n", url)
		},
	}
	if opts.noBrowser {
		service.OpenBrowser = func(string) error { return nil }
	}
	token, err = deps.Authorize(ctx, service)
	if err != nil {
		return identity.Token{}, authError(stderr, err)
	}
	if err := store.Write(ctx, token); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: store token: %v\n", err)
		return identity.Token{}, 1
	}
	return token, 0
}

func onboardInterests(ctx context.Context, cfg config.Config, opts onboardOptions, reader *bufio.Reader, stdout, stderr io.Writer, deps onboardDeps, profiles *sqlite.ProfileRepository, tasteService *taste.Service) int {
	channels, err := deps.Channels(ctx, "owner")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: list subscriptions: %v\n", err)
		return 1
	}
	hasSubscriptions := len(channels) > 0
	_, _ = fmt.Fprintln(stdout, "step 2/2: choose interests")
	printSeedSources(stdout, identity.SeedSources(hasSubscriptions))

	source, code := chooseSource(opts, reader, stdout, stderr, hasSubscriptions)
	if code != 0 {
		return code
	}
	interests, code := chooseInterests(opts, source, channels, reader, stdout, stderr)
	if code != 0 {
		return code
	}
	if err := ensureInterests(ctx, profiles, tasteService, interests, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "interests set: %s\n", strings.Join(interests, ", "))
	_, _ = fmt.Fprintln(stdout, "onboarding complete")
	return 0
}

func printSeedSources(w io.Writer, sources []identity.SeedSourceInfo) {
	for _, info := range sources {
		status := "available"
		if !info.Available {
			status = "unavailable: " + info.Reason
		}
		_, _ = fmt.Fprintf(w, "  %s [%s]\n", info.Source, status)
	}
}

func chooseSource(opts onboardOptions, reader *bufio.Reader, stdout, stderr io.Writer, hasSubscriptions bool) (identity.SeedSource, int) {
	if opts.source != "" {
		source, _, err := identity.SelectSeedSource(opts.source, hasSubscriptions)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "ytfeed onboard: %v\n", err)
			return "", 2
		}
		return source, 0
	}
	if opts.interests != "" && !hasSubscriptions {
		return identity.SeedSubscriptionsOnly, 0
	}
	defaultSource := ""
	for _, info := range identity.SeedSources(hasSubscriptions) {
		if info.Available {
			defaultSource = string(info.Source)
			break
		}
	}
	if defaultSource == "" {
		_, _ = fmt.Fprintln(stderr, "ytfeed onboard: no seed source is available; import subscriptions or pass --interests")
		return "", 2
	}
	answer := readPrompt(reader, stdout, "seed source", defaultSource)
	source, _, err := identity.SelectSeedSource(answer, hasSubscriptions)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: %v\n", err)
		return "", 2
	}
	return source, 0
}

func chooseInterests(opts onboardOptions, source identity.SeedSource, channels []identity.Subscription, reader *bufio.Reader, stdout, stderr io.Writer) ([]string, int) {
	if opts.interests != "" {
		interests, err := taste.NormalizeInterests(splitCSV(opts.interests))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "ytfeed onboard: %v\n", err)
			return nil, 2
		}
		return interests, 0
	}
	if source != identity.SeedSubscriptionsOnly {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: seed source %s does not propose interests yet\n", source)
		return nil, 2
	}

	proposal := proposeFromChannels(channels)
	_, _ = fmt.Fprintf(stdout, "proposed interests: %s\n", strings.Join(proposal, ", "))
	answer := readPrompt(reader, stdout, "interests", strings.Join(proposal, ","))
	interests, err := taste.NormalizeInterests(splitCSV(answer))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed onboard: %v\n", err)
		return nil, 2
	}
	return interests, 0
}

func proposeFromChannels(channels []identity.Subscription) []string {
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].SubscriberCount == channels[j].SubscriberCount {
			return channels[i].Title < channels[j].Title
		}
		return channels[i].SubscriberCount > channels[j].SubscriberCount
	})
	seen := map[string]bool{}
	var out []string
	for _, channel := range channels {
		slug := slugify(channel.Title)
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		out = append(out, slug)
		if len(out) == 5 {
			break
		}
	}
	if len(out) == 0 {
		return append([]string(nil), starterInterests...)
	}
	return out
}

var starterInterests = []string{"ai-agents", "local-llms", "developer-tools", "machine-learning", "distributed-systems"}

func ensureInterests(ctx context.Context, profiles *sqlite.ProfileRepository, tasteService *taste.Service, interests []string, stdout io.Writer) error {
	_, err := tasteService.Interests(ctx, "owner")
	if errors.Is(err, taste.ErrProfileNotFound) {
		profile := provisionalProfile("owner", interests)
		if err := profiles.Save(ctx, profile); err != nil {
			return fmt.Errorf("create profile: %w", err)
		}
		_, _ = fmt.Fprintln(stdout, "using provisional profile rubric=spike-s01")
		return nil
	}
	if err != nil {
		return err
	}
	return tasteService.SetInterests(ctx, "owner", interests)
}

func provisionalProfile(userID string, interests []string) taste.Profile {
	return taste.Profile{
		UserID:        userID,
		Note:          "provisional onboarding profile",
		RubricVersion: "spike-s01",
		Axes: []taste.Axis{
			{Name: "depth", Weight: 0.5, PositiveExamples: []string{"substantive explanation"}, NegativeExamples: []string{"shallow summary"}},
			{Name: "novelty", Weight: 0.5, PositiveExamples: []string{"new angle"}, NegativeExamples: []string{"repeated take"}},
			{Name: "signal-to-noise", Weight: 0.5, PositiveExamples: []string{"dense useful segment"}, NegativeExamples: []string{"padding"}},
			{Name: "practical-value", Weight: 0.5, PositiveExamples: []string{"actionable step"}, NegativeExamples: []string{"abstract talk"}},
		},
		Interests: interests,
	}
}

func readPrompt(reader *bufio.Reader, stdout io.Writer, label, fallback string) string {
	_, _ = fmt.Fprintf(stdout, "%s [%s]: ", label, fallback)
	if reader == nil {
		return fallback
	}
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return fallback
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fallback
	}
	return line
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func slugify(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case !lastDash && b.Len() > 0:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
