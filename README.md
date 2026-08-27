# tross-scraper — LinkedIn Profile API

Give it a LinkedIn profile URL, get the profile back as structured JSON.

**Purely reverse-engineered — no browser, no headless Chrome, no HTML parsing.**
It calls LinkedIn's private JSON API directly over HTTP and unpacks the response
itself. No browser-automation dependency in `go.mod`, no HTML parser in the
codebase. A profile costs **~1 second** and **~30 MB**.

Go + Gin, Redis for caching. The response is **config-driven**: each of the 21
sections can be switched on or off, and nothing is ever invented.

```bash
curl -X POST https://tross-scraper-api.onrender.com/public/v1/profile \
  -H 'content-type: application/json' \
  -d '{"profileUrl": "https://www.linkedin.com/in/williamhgates/"}'
```

<details>
<summary><b>Sample response</b> — note <code>patents: []</code> and
<code>backgroundPhoto: null</code>. Empty is reported as empty.</summary>

```json
{
  "code": "00000",
  "message": "success",
  "result": {
    "profile": {
      "identity": { "name": "Priya Nair", "firstName": "Priya", "lastName": "Nair", "pronouns": null },
      "headline": "Senior Backend Engineer · Distributed Systems",
      "location": "Bengaluru, Karnataka, India",
      "industry": "Financial Services Software",
      "about": "Backend engineer focused on payment rails and reliability at scale.",
      "contactInfo": null,
      "openTo": ["Speaking engagements"],
      "images": { "profilePhoto": "https://media.licdn.com/dms/image/…", "backgroundPhoto": null }
    },
    "experience": [
      { "title": "Senior Backend Engineer", "company": "Razorstack", "employmentType": "Full time",
        "period": { "start": "2022-04", "end": null, "present": true }, "media": [] }
    ],
    "education": [{ "school": "BITS Pilani", "degree": "B.E. Computer Science",
        "period": { "start": "2013", "end": "2017", "present": false }, "media": [] }],
    "skills": [{ "name": "Go", "endorsements": 48 }],
    "patents": [],
    "meta": {
      "publicId": "priya-nair-eng", "cached": false, "fetchedAt": "2026-08-27T15:36:59Z",
      "sectionsReturned": ["experience", "education", "skills", "…"],
      "sectionsUnavailable": ["contactInfo", "featured", "services", "…"]
    }
  }
}
```

</details>

[Setup](#setup) · [Configuration](#configuration) · [API](#api) ·
[Response schema](#response-schema) · [Approach](#approach) ·
[Deployment](#deployment) · [Development](#development) ·
[Known limitations](#known-limitations)

---

## Setup

**You need:** Go 1.24+ and Docker. That's the whole toolchain.

```bash
git clone https://github.com/Tanmay-Tripathi/tross-scraper.git
cd tross-scraper

cp config/local.example.yml config/local.yml   # then add cookies, below
make up                                        # Redis in Docker
make run                                       # API on http://localhost:4201

curl localhost:4201/private/v1/health/ready    # linkedin "up" means you're set
```

Everything in Docker instead: `make stack`.

### Getting the LinkedIn cookies

The API acts as a logged-in user, so it needs two cookies. **Use a throwaway
account**, set to *Settings → Visibility → Profile viewing options → Private
mode*.

> Copying a cookie from DevTools is a one-time manual step, like pasting an API
> key. The running service never launches or drives a browser.

1. Log in in a **normal window** — not incognito, whose cookies vanish on close.
2. DevTools (`F12`) → **Chrome:** Application → Cookies → `https://www.linkedin.com`.
   **Firefox:** Storage → Cookies.
3. Copy **`li_at`** (~200 chars, starts `AQEDA…`). This *is* the login.
4. Copy **`JSESSIONID`**. DevTools shows `"ajax:1234567890"` — **strip the
   quotes**, paste only `ajax:1234567890`.
5. Paste both into `config/local.yml` (gitignored), then **don't click "Log
   out"** — that kills `li_at` server-side. Just close the tab.

```yaml
LinkedIn:
  li_at: "AQEDAS…"
  jsessionid: "ajax:1234567890"
```

Verify before anything else — this checks the session first and stops if the
cookies are stale:

```bash
go run ./cmd/spike https://www.linkedin.com/in/williamhgates/
```

It calls every endpoint, saves raw responses to `testdata/fixtures/` (gitignored
— real people's data), and prints which sections came back.

---

## Configuration

YAML, chosen with `-config` (default `./config/local.yml`).

| File | Purpose | Committed |
|---|---|---|
| `config/local.example.yml` | template | yes |
| `config/local.yml` | your config, with real cookies | **no** |
| `config/production.yml` | deployed config — every secret is a `${VAR}` | yes |

Values support `${VAR}` and `${VAR:-default}`, resolved from the environment at
startup. **That is how credentials stay out of this repo:** the committed file
names the variable, the deployment supplies the value. Config is validated at
boot, so a missing Redis host or a typo'd section name stops the process rather
than surfacing on the first request.

All 21 sections default to on:

```yaml
Sections:
  patents: false      # this key will not appear in any response
```

| Environment variable | Required | Default | Purpose |
|---|---|---|---|
| `REDIS_HOST` | yes | — | Redis hostname |
| `LINKEDIN_LI_AT` | yes | — | the `li_at` session cookie |
| `LINKEDIN_JSESSIONID` | yes | — | the `JSESSIONID` value, no quotes |
| `REDIS_PORT` | no | `6379` | Redis port |
| `REDIS_PASSWORD` / `REDIS_USERNAME` | no | empty | Redis auth |
| `REDIS_TLS_ENABLED` | no | `false` | TLS to Redis |
| `PORT` | no | `4201` | HTTP listen port |
| `APP_ENV` | no | `prd` | `local` / `stg` / `uat` / `prd` |
| `LOG_LEVEL` | no | `info` | `debug` / `info` / `warn` / `error` |
| `CORS_ALLOWED_ORIGINS` | no | empty | comma-separated browser origins |
| `LINKEDIN_DAILY_BUDGET` | no | `200` | hard cap on live scrapes per day |
| `LINKEDIN_CACHE_TTL_MINUTES` | no | `360` | how long a profile stays cached |
| `OTLP_EXPORTER_URL` | no | empty | OTLP endpoint; empty disables tracing |

---

## API

`/public/v1` is unauthenticated and safe to expose. `/private/v1` holds probes
and should not be routed publicly.

### `POST /public/v1/profile`

```json
{
  "profileUrl": "https://www.linkedin.com/in/williamhgates/",
  "sections": { "patents": false },
  "refresh": false
}
```

| Field | Required | Meaning |
|---|---|---|
| `profileUrl` | yes | any normal profile URL — `www` or not, country subdomain, trailing slash, tracking params, sub-page |
| `sections` | no | overrides the configured default, one section at a time |
| `refresh` | no | bypasses the cache. Still counts against the daily budget |

**Errors** — every failure returns `{ "code", "message" }` at its own status:

| Code | HTTP | Meaning |
|---|---|---|
| `PRF01` | 400 | not a LinkedIn member profile URL |
| `PRF02` | 404 | profile missing, private, or invisible to our account |
| `PRF03` | 502 | LinkedIn answered in a shape we can't parse |
| `PRF04` | 503 | **our session expired** — needs fresh cookies |
| `PRF05` | 429 | LinkedIn is rate limiting us |
| `PRF06` | 400 | unknown section name |
| `PRF07` | 429 | our own daily budget is spent |
| `IR01` | 400 | malformed request body |

`PRF04` gets its own code because it's the one failure that needs a human. It
must never look like a generic 500.

**Health:**

| Endpoint | What it does |
|---|---|
| `GET /public/v1/health` | liveness — touches nothing, cheap for an uptime monitor |
| `GET /private/v1/health/ready` | probes Redis **and the LinkedIn session**; 503 if one is down |
| `GET /metrics` | Prometheus: request counts, latency, in-flight |

Readiness is why an expired cookie shows on a dashboard instead of being found by
a caller. `database` reports `disabled` — see [Storage](#storage).

**Bruno collection** — runnable examples with assertions in
[`api_collection/`](api_collection/). Open the folder in
[Bruno](https://www.usebruno.com/), pick an environment, hit Run.
`scrape-profile-sections-override` proves a disabled section is *absent*, not
empty.

---

## Response schema

`{ code, message, result }`. Inside `result`: a `profile` object, one key per
enabled section, and `meta`.

**The three rules:**

| Situation | Result |
|---|---|
| enabled, has data | the data |
| enabled, no data | `[]` for a list, `null` for a single value |
| disabled | **the key is absent entirely** |

That last line is the whole design: a caller can always tell *"has no patents"*
from *"didn't ask for patents"*. `meta.sectionsUnavailable` names every section
that was enabled but came back empty, so `[]` is never ambiguous.

**Dates keep LinkedIn's precision.** A year with no month is `"2022"`, never
`"2022-01"` — inventing January is exactly the fabrication this API refuses. An
ongoing role is `"end": null, "present": true`.

**The 21 sections** — the first three nest under `profile`, the rest are
top-level arrays:

`about` · `contactInfo` · `openTo` · `experience` · `education` · `skills` ·
`featured` · `projects` · `licensesAndCertifications` · `courses` ·
`recommendations` · `volunteerExperience` · `publications` · `patents` ·
`honorsAndAwards` · `testScores` · `languages` · `organizations` · `services` ·
`careerBreaks` · `causes`

---

## Approach

**LinkedIn's page is a shell.** The HTML holds almost no profile data — the
browser loads a shell, then calls LinkedIn's private JSON API ("Voyager") and
draws the page from that. This service skips the page and calls Voyager directly.
**That's the reverse engineering.** One request does most of the work:

```
GET /voyager/api/identity/dash/profiles?q=memberIdentity&memberIdentity={publicId}
```

with the `FullProfileWithEntities` decoration — ~120 KB, 100+ entities.

**The old Voyager is dying.** Legacy `/identity/profiles/{publicId}/…` routes now
answer **410 Gone**, one at a time rather than all at once. The dash profile
covers everything they carried except contact info, which no reachable endpoint
exposes any more — so `contactInfo` is always `null` and listed in
`meta.sectionsUnavailable`.

**Authenticating** takes four things per request:

| What | Value |
|---|---|
| `li_at` cookie | the session — this *is* the login |
| `JSESSIONID` cookie | `"ajax:1234567890"`, **with** quotes |
| `csrf-token` header | the same value, **without** quotes |
| `x-restli-protocol-version` | `2.0.0` — selects the modern format |

That asymmetry is two traps in one. The quotes differ between the two headers —
and a literal `"` is an invalid cookie byte, so `net/http` silently strips it and
sends the cookie unquoted anyway. `http.Cookie{Quoted: true}` is what actually
puts quotes on the wire. Get it wrong and LinkedIn answers `CSRF check failed`,
or kills the session outright.

**Parsing is two steps.** Voyager returns a flat envelope, not nested JSON:
`data` refers to things by `entityUrn`, and `included` is a flat list of every
object. So index `included`, then follow the pointers out of `data`. If `data`
omits a section whose entities sit in `included` anyway, the parser falls back to
scanning by type — without that, an upstream change would silently empty a
section, the worst kind of failure because nobody notices.

Images are indirect too: `rootUrl` plus the largest artifact's path. Missing
either half gives `null`, never a constructed link that 404s.

**Why no headless browser** — besides it being a requirement:

| | Voyager API | Headless browser |
|---|---|---|
| Speed | ~1 s | 5–15 s |
| Memory | ~30 MB | ~1 GB — won't fit free hosting |
| Output | structured JSON | HTML we'd have to guess at |
| Breaks when | LinkedIn changes its API | LinkedIn changes any CSS class |

### Protecting the account

Aggressive scraping gets an account restricted, and a restricted account is a
dead demo.

| Guard | What it does |
|---|---|
| Redis cache (6 h) | a profile is fetched once, not once per request |
| Daily budget | hard ceiling; `refresh` counts against it too |
| One reused session | a single cookie jar, like one browser tab |
| Real browser headers | a bare Go client is trivially detectable |
| Small jitter | avoids a machine-perfect request rhythm |
| Private viewing mode | reads stay anonymous |

**The full profile is cached, then filtered** — two callers wanting different
sections of the same person cost one scrape, not two.

### Storage

**Redis is the only datastore, deliberately.** The service holds no relational
state — fetch, map, cache, return. Redis does two jobs: the profile cache, and a
self-expiring per-day counter for the scrape budget.

A Postgres that nothing queries would be one more thing to provision, one more
thing to fail at boot, and — on a free tier that expires managed databases after
a month — a live demo dying on a timer for a dependency the code never used. The
seam remains (`RepositoryAccess.Db` is a nil `*db.Store`), so wiring one back in
is a change in `cmd/app/app.go` and nowhere else. Readiness says `disabled`, not
`down`: absent-by-design isn't a fault.

### Blast radius and layering

```
route → controller → service → repository / client → Redis / LinkedIn
```

Everything about LinkedIn's wire format lives in `pkg/linkedin/voyager`, so a
change like those 410s moves one package. Controllers own HTTP and never see
LinkedIn; services never see a Gin type; the client returns typed errors, so no
layer above reads an HTTP status. And the rule that nothing is fabricated lives
in exactly one function, `pkg/mapper.ToProfileResult` — one place makes it
checkable rather than merely intended, and tests assert the three rules there.

---

## Deployment

A static binary in an Alpine image, non-root. One service, one Dockerfile, plus
Redis.

**Render** — [`render.yaml`](render.yaml) provisions both over HTTPS. Push to
GitHub, then Render → **New → Blueprint** → select the repo, and set
`LINKEDIN_LI_AT` and `LINKEDIN_JSESSIONID` on the API service. They're marked
`sync: false`, so Render prompts once and stores them encrypted. That's the whole
deploy. Health check path: `/public/v1/health`.

**Any Docker host:**

```bash
docker build -t tross-scraper .
docker run -p 4201:4201 \
  -e REDIS_HOST=redis-host \
  -e LINKEDIN_LI_AT="AQEDAS…" \
  -e LINKEDIN_JSESSIONID="ajax:1234567890" \
  tross-scraper
```

**Refreshing cookies** — an expired `li_at` gives `PRF04` and `linkedin: down` on
readiness. Repeat the DevTools steps, update the env var, restart. About a
minute, no redeploy, because the value was never baked into the image.

---

## Development

```bash
make lint          # go fmt + go mod tidy + go vet
make test          # go test ./...
make build         # static binary
```

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs gofmt, vet,
build and tests, then builds the Docker image.

**The test suite needs no LinkedIn account** — parsing is covered by synthetic
payloads in the exact Voyager shape, and the contract rules are asserted against
the mapper directly.

Conventions are in [CLAUDE.md](CLAUDE.md); the design and build plan is in
[TECH_PLAN.md](TECH_PLAN.md).

---

## Known limitations

**1. Every call is made *as* the authenticated account.** LinkedIn returns
exactly what that account would see in a browser — no more. So *"can we scrape
profile X?"* reduces to *"can our account see profile X?"* A direct consequence:
**the same URL returns different data for different accounts**, so our output
won't match what *you* see on linkedin.com.

Reliable for anyone: name, headline, location, industry, about, experience,
education, skills, certifications, projects, publications, patents, honours,
courses, languages, organizations, volunteering. These are gated by relationship
and are often empty — **not a parser bug**:

| Field | Why |
|---|---|
| `contactInfo` | the endpoint is gone (410); also 1st-degree only |
| `openTo` | often shown only to recruiters or connections |
| `images.profilePhoto` | the member can restrict it to connections |
| `identity.name` | some members display only a surname initial |

**2. Reading a profile is visible to its owner.** We call the endpoint the
browser uses to render a profile page, so a fetch most likely registers as a
profile view. Use **Private (anonymous) viewing mode**, and a throwaway account.

**3. Cookies expire.** A password change or security prompt kills the session;
the API returns `PRF04` until it's refreshed.

**4. `featured`, `services`, `careerBreaks` return `[]`.** They live behind newer
endpoints that aren't wired up. Returned empty rather than omitted, and listed in
`meta.sectionsUnavailable`, so the response never implies the profile has none.
`causes` is derived from volunteering entries — the only honest source available.

**5. Voyager is private and undocumented.** LinkedIn can change it without
warning. The quarantined package limits the damage, but a change means a fix.

**6. Rate limits are real and undocumented.** Hence the cache and the daily cap.

**7. Free-tier cold starts.** Render sleeps idle services, so the first request
after a quiet period is slow.

**8. No auth or per-IP rate limiting on the API itself.** The daily scrape budget
is what protects the account today.

**9. Not for bulk harvesting.** Single-profile, on-demand, deliberately capped.

**Legal note:** scraping LinkedIn while logged in is against LinkedIn's User
Agreement, and the account carries a genuine risk of restriction. This project
was built for a technical assignment that explicitly permits using your own
credentials. Use a throwaway account.

---

## License

MIT — see [LICENSE](LICENSE).
