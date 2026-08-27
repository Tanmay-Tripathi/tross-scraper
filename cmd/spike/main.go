// Command spike calls every Voyager endpoint for a set of profiles, saves the raw
// responses as fixtures, and reports which sections are reachable.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/linkedin/voyager"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/network"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/validation"
)

func main() {
	configPath := flag.String("config", "./config/local.yml", "path to the config file")
	outDir := flag.String("out", "./testdata/fixtures", "directory to write raw responses into")
	pause := flag.Duration("pause", 3*time.Second, "delay between profiles, to stay human-paced")
	// Off by default: /voyager/api/me appears to invalidate the browser session that
	// minted these cookies, so no run makes that call unless it is asked for.
	checkSession := flag.Bool("check-session", false, "probe /me first to verify the cookies (this logs the browser session out)")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/spike [-config path] [-out dir] <profile-url>...")
		os.Exit(2)
	}

	if err := run(*configPath, *outDir, *pause, *checkSession, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "\nfatal: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath, outDir string, pause time.Duration, checkSession bool, urls []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if !cfg.LinkedIn.Configured() {
		return fmt.Errorf("no LinkedIn cookies in %s — add the LinkedIn block with li_at and jsessionid", configPath)
	}

	logger := log.New(log.LogConfig{
		ServiceName: cfg.AppName,
		AppEnv:      cfg.Environment,
		AppVersion:  cfg.AppVersion,
		Level:       "warn", // the printed report is the output, not the logs
	})

	netOps, err := network.NewNetworkOps("linkedin", logger, network.Options{
		Timeout:         time.Duration(cfg.LinkedIn.RequestTimeoutSeconds) * time.Second,
		EnableCookieJar: true,
	})
	if err != nil {
		return err
	}

	client, err := voyager.NewClient(netOps, voyager.Credentials{
		LiAt:       cfg.LinkedIn.LiAt,
		JSessionID: cfg.LinkedIn.JSessionID,
		UserAgent:  cfg.LinkedIn.UserAgent,
	}, logger)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if checkSession {
		fmt.Println("checking the session is alive…")
		if err := client.SessionValid(ctx); err != nil {
			return fmt.Errorf("session check failed — cookies are probably stale: %w", err)
		}
		fmt.Print("session OK\n\n")
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create fixture directory: %w", err)
	}

	coverage := newCoverage()

	for i, rawURL := range urls {
		publicID, appErr := validation.ParseProfileURL(rawURL)
		if appErr != nil {
			fmt.Printf("skipping %q: not a profile URL\n\n", rawURL)
			continue
		}

		if i > 0 {
			time.Sleep(pause)
		}

		if err := probe(ctx, client, publicID, outDir, coverage); err != nil {
			return err
		}
	}

	coverage.report()
	fmt.Printf("\nfixtures written to %s\n", outDir)
	return nil
}

// endpointPause spaces out the calls within one profile, for the same reason
// -pause spaces out the profiles: a machine-regular burst is an easy tell.
const endpointPause = 1500 * time.Millisecond

// probe calls every endpoint for one profile and records what came back.
func probe(ctx context.Context, client *voyager.Client, publicID, outDir string, cov *coverage) error {
	fmt.Printf("=== %s ===\n", publicID)

	profileDir := filepath.Join(outDir, sanitize(publicID))
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return fmt.Errorf("create fixture directory: %w", err)
	}

	// Every endpoint is probed on its own. FetchAll stops at the first essential
	// failure, which is correct in production but leaves a spike blind to the rest.
	for i, endpoint := range voyager.ProfileEndpoints {
		if i > 0 {
			time.Sleep(endpointPause)
		}

		var query map[string]string
		if endpoint.Query != nil {
			query = endpoint.Query(publicID)
		}

		body, err := client.Get(ctx, endpoint.Path(publicID), query)
		if err != nil {
			fmt.Printf("  %-16s FAILED   %v\n", endpoint.Name, err)
			cov.recordFailure(endpoint.Name)
			continue
		}

		path := filepath.Join(profileDir, endpoint.Name+".json")
		if err := os.WriteFile(path, prettify(body), 0o644); err != nil {
			return fmt.Errorf("write fixture %s: %w", path, err)
		}

		graph, err := voyager.NewGraph(body)
		if err != nil {
			fmt.Printf("  %-16s %6d bytes  (not the normalized shape: %v)\n", endpoint.Name, len(body), err)
			cov.recordFailure(endpoint.Name)
			continue
		}

		fmt.Printf("  %-16s %6d bytes  %3d entities\n", endpoint.Name, len(body), graph.Size())
		cov.recordSuccess(endpoint.Name, graph.Types())
	}

	fmt.Println()
	return nil
}

// coverage accumulates what the probes found.
type coverage struct {
	ok      map[string]int
	failed  map[string]int
	types   map[string]int
	sources map[string]map[string]bool // entity type -> endpoints that carried it
}

func newCoverage() *coverage {
	return &coverage{
		ok:      map[string]int{},
		failed:  map[string]int{},
		types:   map[string]int{},
		sources: map[string]map[string]bool{},
	}
}

func (c *coverage) recordSuccess(endpoint string, types map[string]int) {
	c.ok[endpoint]++
	for typeName, count := range types {
		c.types[typeName] += count
		if c.sources[typeName] == nil {
			c.sources[typeName] = map[string]bool{}
		}
		c.sources[typeName][endpoint] = true
	}
}

func (c *coverage) recordFailure(endpoint string) { c.failed[endpoint]++ }

// report prints which endpoints worked and which entity types were seen.
func (c *coverage) report() {
	fmt.Println("=== endpoint results ===")
	for _, endpoint := range voyager.ProfileEndpoints {
		fmt.Printf("  %-16s ok=%d failed=%d   feeds: %s\n",
			endpoint.Name, c.ok[endpoint.Name], c.failed[endpoint.Name],
			strings.Join(endpoint.Sections, ", "))
	}

	fmt.Println("\n=== entity types seen (this is the ground truth for mapping) ===")
	names := make([]string, 0, len(c.types))
	for name := range c.types {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return c.types[names[i]] > c.types[names[j]] })

	for _, name := range names {
		from := make([]string, 0, len(c.sources[name]))
		for endpoint := range c.sources[name] {
			from = append(from, endpoint)
		}
		sort.Strings(from)
		fmt.Printf("  %4d  %-62s %s\n", c.types[name], shortType(name), strings.Join(from, ","))
	}
}

// shortType drops LinkedIn's namespace so the report stays readable.
func shortType(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 && i < len(name)-1 {
		return name[i+1:]
	}
	return name
}

func prettify(body []byte) []byte {
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return body
	}
	pretty, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return body
	}
	return pretty
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
}
