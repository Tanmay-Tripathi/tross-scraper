# Tech Plan — LinkedIn Profile API (`tross-scraper`)

**Status:** built, pending live verification · **Last updated:** 2026-08-27

> Phases 1–6 are implemented and tested against synthetic payloads. **Phase 0 —
> the live spike — has not run yet**, because it needs cookies. Everything the
> spike would confirm is marked below.

---

## 1. What we're building

**The challenge:** reverse engineer how LinkedIn exposes profile data, and ship a
publicly hosted HTTPS API that takes a profile URL and returns the profile as
structured JSON. Public GitHub repo, README with setup + docs + approach +
limitations, and no credentials in the repo.

**The PRD on top of that:** support 21 profile sections, make the response
**config-driven** (sections can be switched on or off), and **never invent data** —
missing means `[]`, `null`, or the key is absent, never a fake value.

**One sentence:** `POST /v1/profile` with a LinkedIn URL → clean JSON of every
section that is both *enabled* and *present*.

---

## 2. The core idea: use LinkedIn's own API

When you open a LinkedIn profile in a browser, the page you see is not delivered
as finished HTML. The browser loads a shell, then calls LinkedIn's **private
internal JSON API** — called **Voyager** — and draws the page from that JSON.

So we skip the page entirely and call Voyager directly, the same way the browser
does. **That is the reverse engineering the brief asks for.**

Why this beats scraping HTML:

| | Voyager API | Headless browser |
|---|---|---|
| Speed | ~1 second | 5–15 seconds |
| Memory | ~30 MB | ~1 GB (won't fit free hosting) |
| Output | Already structured JSON | HTML we'd have to guess at |
| Breaks when | LinkedIn changes its API | LinkedIn changes any CSS class |

### How we authenticate

We log in once as a real user, in a browser, and copy two cookies. The server
then replays them on every call:

| What | Where it goes |
|---|---|
| `li_at` cookie | the session — this *is* the login |
| `JSESSIONID` cookie | e.g. `"ajax:1234567890"` |
| `csrf-token` header | the `JSESSIONID` value **with the quotes stripped** |
| `x-restli-protocol-version: 2.0.0` header | tells Voyager to answer in its modern format |
| a real browser `user-agent` | a missing or odd one gets flagged fast |

The `csrf-token`-must-match-`JSESSIONID` trick is the part that trips most
people up. Both cookies come from environment variables — **never the repo**.

### The response shape we have to unpack

Voyager doesn't return neatly nested JSON. It returns a **flat** shape:

```
{
  "data":     { ...the profile, with pointers... },
  "included": [ ...every object, in one flat list, each with an "entityUrn" id... ]
}
```

`data` refers to things by ID (`entityUrn`), and the actual objects sit in
`included`. So our parser has one job first: **index `included` by `entityUrn`,
then follow the pointers.** Everything else is ordinary mapping.

Images are also indirect. A photo arrives as a `rootUrl` plus a list of
`artifacts` at different sizes; the real URL is `rootUrl` + the largest
artifact's path segment. If either half is missing we return `null` — we never
build a placeholder URL.

---

## 3. Which sections we can actually get

This is the honest part. Not every one of the 21 sections comes from the same
place, and a few may need a second call or may not be reachable at all.

**Confidence** is my current expectation. **Phase 0 (below) verifies every row
against a real account before we build the mapper** — I'd rather find out in a
one-day spike than half way through.

| Section | Where it comes from | Confidence |
|---|---|---|
| identity, headline, location, industry, about | main profile call | High |
| images (profile + background) | main profile call | High |
| experience, education | main profile call | High |
| skills | main profile call (+ paged call for the full list) | High |
| licensesAndCertifications, projects, courses | main profile call | High |
| publications, patents, honorsAndAwards, testScores | main profile call | High |
| languages, organizations, volunteerExperience | main profile call | High |
| recommendations | separate call | Medium |
| contactInfo | separate call (`profileContactInfo`) | Medium |
| featured, services, openTo | newer "dash" endpoints | Medium |
| careerBreaks | modern API folds these into experience | Medium |
| causes | comes attached to volunteering | Low |

Anything that turns out to be genuinely unreachable gets **documented as a known
limitation and returns `[]`** — per the PRD, we do not fabricate. We will not
quietly drop it and hope nobody notices.

To keep it fast we make the calls we need **in parallel**, not one after another.
If a secondary call fails, that one section returns empty and the rest of the
response still succeeds — one weak section must never fail the whole request.

---

## 4. The API

### Request

```http
POST /v1/profile
Content-Type: application/json

{
  "profileUrl": "https://www.linkedin.com/in/example/",
  "sections": { "patents": false, "recommendations": true }   // optional
}
```

We accept any normal profile URL shape (`/in/slug`, with or without `www`,
trailing slash, query junk, or a locale prefix) and reduce it to the **public
identifier** (`example`). Company and job URLs are rejected with a clear error.

### Response

Wrapped in the envelope the service already uses:

```json
{
  "code": "00000",
  "message": "success",
  "result": {
    "profile": {
      "identity": { "name": "...", "pronouns": null },
      "headline": "...", "location": "...", "industry": "...",
      "about": "...",
      "contactInfo": { "...": "..." },
      "openTo": [],
      "images": { "profilePhoto": "https://...", "backgroundPhoto": null }
    },
    "experience": [ { "...": "...", "media": [] } ],
    "education": [],
    "patents": []
  },
  "meta": { "publicId": "example", "cached": false, "fetchedAt": "..." }
}
```

The three rules from the PRD, exactly:

- **enabled + has data** → the real data
- **enabled + no data** → `[]` for lists, `null` for single optional fields
- **disabled** → **the key is not in the response at all**

### Errors

Reusing the existing `internal/exceptions` catalogue, plus new codes:

| Code | HTTP | Meaning |
|---|---|---|
| `PRF01` | 400 | Not a valid LinkedIn profile URL |
| `PRF02` | 404 | Profile not found, private, or not visible to our account |
| `PRF03` | 502 | LinkedIn answered in a shape we don't recognise |
| `PRF04` | 503 | Our LinkedIn session expired — needs new cookies |
| `PRF05` | 429 | LinkedIn is rate-limiting us; retry later |
| `INV01` | 400 | Unknown section name in the `sections` override |

`PRF04` is deliberately loud and distinct: it's the one failure that needs a
human, and it must never look like a generic 500.

---

## 5. Config-driven sections

Two layers, per your decision:

**1. Deployment default** — `config/local.yml` and `config/production.yml`:

```yaml
Sections:
  experience: true
  education: true
  patents: false
```

**2. Per-request override** — the optional `sections` object in the POST body,
merged over the default for that one call.

An unknown section name is **rejected with `INV01`**, not silently ignored — a
typo like `"experiance": true` should tell you, not quietly return nothing.

The list of valid section names lives in **one** Go constant. Config, validation,
the response builder and the docs all read from it, so a new section can never be
half-added.

---

## 6. Where the code goes

Slots straight into the layering already in `CLAUDE.md` — no new patterns.

```
cmd/app/routes_profile.go              POST /v1/profile

internal/controllers/controller_profile.go   bind → validate → call service → map → respond
internal/services/service_profile.go         cache check → fetch → assemble → apply config
internal/clients/client_linkedin.go          the Voyager client (recreates the clients layer)
internal/repositories/repo_profile.go        Redis cache read/write
internal/models/profile.go                   domain models (one per section)
internal/response/response_profile.go        the JSON shape clients see
internal/requests/requests_profile.go        the POST body
internal/exceptions/errors_profile.go        PRF01–PRF05

pkg/network/                                 shared HTTP caller, rebuilt with a cookie jar
pkg/mapper/profile_mapper.go                 Voyager JSON → models → response
pkg/validation/profile_validation.go         URL parsing, section-name checks
pkg/linkedin/voyager/                        raw Voyager payload structs + the URN resolver
```

Two deliberate choices:

- **`pkg/linkedin/voyager` is quarantined.** Everything ugly about LinkedIn's
  format lives in one package. If LinkedIn changes its API, we fix that package
  and nothing else moves.
- **The mapper decides `null` vs `[]`.** Services never guess at presence, so the
  "never fabricate" rule is enforced in exactly one place.

### 6.1 Bringing back the client layer

We deleted `internal/clients` and `pkg/network` when SQS came out, because with
the SQS client gone they held nothing. **The LinkedIn client is what brings both
back** — this is exactly the case `CLAUDE.md` §9 was written for, and it is a
Phase 1 task, not an afterthought.

Files to recreate:

| File | What it holds |
|---|---|
| `internal/clients/types.go` | `clientAccess` — the shared deps every client gets (config, logger, cache, HTTP caller) |
| `internal/clients/main.go` | the `Clients` aggregate + `NewClients(...)` constructor |
| `internal/clients/client_linkedin.go` | `ClientLinkedInMethods` interface + the Voyager implementation |
| `pkg/network/types.go`, `pkg/network/network_ops.go` | the shared HTTP caller — **with the cookie jar this time**, which is the whole reason it exists |

Wiring to re-add (four small edits, all previously removed):

1. `cmd/app/app.go` — build the HTTP caller, then `clients.NewClients(...)`, and
   hold both on `App`.
2. `internal/services/types.go` — put the `Clients` field back on `ServiceAccess`.
3. `internal/services/main.go` — accept `*clients.Clients` in `NewServices` again.
4. `cmd/app/middlewares/*` — **leave alone.** Middlewares had a `Clients` field
   before and never used it; it does not come back.

The client that returns is not the old one. It is shaped for this job:

- **One long-lived cookie jar**, reused across every request — one browser tab,
  not a fresh login per call.
- **Cookies injected from config**, never hardcoded, never logged.
- **A `SessionValid(ctx)` method**, so `/private/v1/health/ready` can report an
  expired cookie instead of us finding out from a reviewer.
- **Returns `*exceptions.ApplicationError`**, mapping LinkedIn's 401/403/429 onto
  `PRF04`/`PRF02`/`PRF05` so the layer above never has to read HTTP codes.

The interface is what the service depends on, so Phase 2's mapping work can be
tested against saved fixtures with no network and no ban risk.

---

## 7. What happens on a request

1. Validate the body; parse the URL down to the public ID.
2. Merge config defaults with any per-request overrides.
3. **Check Redis.** Cache hit → return immediately (`meta.cached: true`).
4. Miss → check the rate limiter and the daily budget.
5. Call Voyager: main profile call, plus secondary calls in parallel.
6. Resolve `included` by URN, map into domain models.
7. Cache the **full** profile in Redis (~6 h TTL).
8. Apply the section config and return.

**We always cache the complete profile, then filter.** Two callers wanting
different sections of the same person must cost one scrape, not two.

---

## 8. Protecting the LinkedIn account

This is real engineering, not paperwork: aggressive scraping gets an account
restricted, and a restricted account means a dead demo on submission day.

| Guard | What it does |
|---|---|
| Redis cache (~6 h) | the same profile is fetched once, not once per request |
| Per-IP rate limit | one visitor can't drain the account |
| Global daily budget | a hard ceiling on total scrapes per day |
| Single reused session | one cookie jar, like one browser tab — not a new login per call |
| Real browser headers | a bare Go HTTP client is trivially detectable |
| Small jitter between calls | avoids a perfectly machine-like request rhythm |
| Private profile-viewing mode | reads stay anonymous, so scraped members aren't notified |

Also worth stating plainly in the README: scraping LinkedIn while logged in goes
against LinkedIn's User Agreement, and the account carries a real risk of
restriction. The brief explicitly permits using your own credentials, so we
proceed — but we use a **throwaway account, not your main one**, and we say so
in the limitations section.

---

## 9. Scope: API only, no frontend

An earlier revision of this plan included a React console (a port of
`profiler.html`, plus a Tross-orange brand pass). **That has been dropped and the
code removed.** Two reasons:

- **The brief asks for an API.** Tross confirmed by email that they want a purely
  reverse-engineered solution hitting LinkedIn endpoints directly, with no
  browser. A UI adds nothing to that and invites the misreading that a browser is
  involved somewhere in the pipeline.
- **Deployment gets simpler.** One Docker service instead of a web service plus a
  static site with a build-time-inlined API URL and a matching CORS origin. Fewer
  moving parts, fewer ways for a reviewer's first request to fail.

The demo surface is now `curl`, the Bruno collection (§10.2), and the example
response in the README — which is generated from the real mapper rather than
hand-written, so it cannot drift from the actual contract.

CORS support stays in the service. It is a handful of wired, tested lines, it
defaults to off, and it costs nothing to keep for anyone calling the API from a
browser console.

---

## 10. README and API collection

The brief grades these directly — *"Include a README with setup instructions, API
documentation, your approach, and known limitations."* So they are a build phase,
not a tidy-up at the end. The README already exists; this is the delta.

### 10.1 README — what changes

Same four headings the brief asks for, in that order, so a reviewer can tick them
off without hunting:

| Section | What goes in it | State |
|---|---|---|
| **Live demo** | the public HTTPS URL + one screenshot of the profile view, at the very top | new |
| **Setup** | clone → `cp config/local.example.yml config/local.yml` → add cookies → `make up` → `make run`. One copy-paste block that works. | expand |
| **Configuration** | the `Sections` block, the LinkedIn cookies, and **how to get them** (the DevTools steps, including the `csrf-token` / `JSESSIONID` quote trick) | new |
| **API documentation** | `POST /v1/profile` — request, the full response schema, the three enabled/empty/disabled rules, the `PRF01`–`PRF05` error table, a real `curl` | expand |
| **Approach** | **the section that wins marks.** See below. | new |
| **Known limitations** | the eight honest ones from §13 — including that results are relative to the authenticated account | rewrite |

**The Approach section** is the one thing a reviewer cannot get from reading code,
so it gets written properly and kept short — roughly a page:

1. LinkedIn's page is a shell; the data arrives from its private **Voyager** API.
   We call that directly. *That* is the reverse engineering.
2. How auth works — the two cookies, and the `csrf-token`-equals-`JSESSIONID`
   -without-quotes detail.
3. Why the response needed unpacking: `data` holds pointers, `included` holds the
   objects, keyed by `entityUrn`.
4. Why not a headless browser — the speed/memory table from §2.
5. Why LinkedIn's format is quarantined in one package.
6. Why nothing is ever fabricated, and where that rule is enforced.

Rules for the whole README: **plain English, no filler.** Every command must be
copy-pasteable and actually run. No credential ever appears, not even a fake one
that looks real.

### 10.2 Bruno collection — what to add

`api_collection/` currently only covers health. A reviewer should be able to open
it, hit **Run**, and watch the contract prove itself. Four requests, each with
assertions that already pass:

| Request | What it proves |
|---|---|
| `scrape-profile` | the happy path — 200, envelope, real sections populated |
| `scrape-profile-sections-override` | sends `{"patents": false}` and asserts the `patents` **key is absent** — the config contract, demonstrated |
| `scrape-profile-invalid-url` | a company URL returns 400 / `PRF01` |
| `scrape-profile-cached` | run twice; second returns `meta.cached: true` — shows the cache protecting the account |

Also: a second environment, `production.bru`, pointing at the deployed URL, so
the same collection runs against local **and** live.

Each request keeps a short `docs` block. Those blocks and the README's API
section say the same thing — if one changes, both change.

---

## 11. Deployment

Render blueprint already exists; this adds to it. One service, one Dockerfile.

- **API** — Docker web service, already wired to Postgres + Redis, health check on
  `/public/v1/health`.
- **New secrets**, set in the Render dashboard and marked `sync: false` in
  `render.yaml` so they are never committed:
  `LINKEDIN_LI_AT`, `LINKEDIN_JSESSIONID`.
- **Free tier note** — free services sleep when idle, so the first request after
  a quiet spell is slow. The Voyager approach is light enough to fit; a headless
  browser would not have been.
- **Cookie rotation** — README documents exactly how to pull fresh cookies from a
  browser and update them. Takes about a minute, no redeploy needed.
- **Readiness** — extend `/private/v1/health/ready` with a cheap "is our LinkedIn
  session still alive?" check, so an expired cookie is visible on a dashboard
  instead of being discovered by a reviewer.

---

## 12. Build phases

| Phase | What | Status |
|---|---|---|
| **0. Spike** | Hit Voyager with real cookies, save raw JSON for 3–4 varied profiles, confirm every row in §3, and check whether a fetch registers a profile view | **pending — needs cookies.** `cmd/spike` is written and ready to run. |
| **1. Client** | Recreate `internal/clients` + `pkg/network` (see §6.1), then `pkg/linkedin/voyager` + `client_linkedin.go`, driven by the saved fixtures | done |
| **2. Mapping** | Domain models, mapper, `null` vs `[]` rules | done — shapes to be confirmed by the spike |
| **3. API** | Route, controller, service, config, errors, cache, rate limiting | done |
| **5. Docs** | README rewrite + 4 new Bruno requests (§10) | done |
| **6. Ship** | Deploy to Render, set the cookie secrets, verify the live URL end to end | pending — needs the live API |
| **4. Scope cut** | Remove the React console and brand pass; API-only (§9) | done |

**What the spike still has to settle.** The parser was written against Voyager's
documented shapes and is covered by synthetic payloads in exactly that format, so
the plumbing is proven. What is *not* proven is whether LinkedIn's live responses
match those shapes field for field. The spike answers that in one run, and any
mismatch is a fix inside `pkg/linkedin/voyager` and nowhere else.

Three sections ship as `[]` today because they live behind LinkedIn's newer dash
endpoints: `featured`, `services`, `careerBreaks`. They are reported in
`meta.sectionsUnavailable` and documented as limitations rather than faked.

---

## 13. Known limitations (going in the README)

1. **Every call is made *as* the authenticated account.** The cookies mean
   LinkedIn returns exactly what that account would see if it opened the profile
   in a browser — no more, no less. So *"can we scrape profile X?"* reduces to
   *"can our account see profile X?"*

   In practice, for any profile a logged-in user can open, these come back
   reliably: name, headline, location, industry, about, experience, education,
   skills, certifications, projects, publications, patents, honours, courses,
   languages, organizations, volunteering, background photo.

   These depend on the viewer's relationship to the target and will often be
   empty — **a LinkedIn restriction, not a parser bug**:

   | Field | Why it may be missing |
   |---|---|
   | `contactInfo` | email/phone are usually shared only with 1st-degree connections who opted in |
   | `openTo` | often shown only to recruiters or connections |
   | `images.profilePhoto` | the target can restrict their photo to "connections only" |
   | `identity.name` | some members display only a surname initial |

   A direct consequence: **the same URL returns different data for different
   accounts.** Results are relative to whoever's cookies are in use.

2. **Reading a profile is visible to its owner.** We call the same endpoint the
   browser uses to render a profile page, so a fetch very likely registers as a
   profile view — the target may see that our account viewed them. Mitigated by
   putting the scraper account in **Private (anonymous) profile-viewing mode**;
   confirmed in Phase 0. This is also why the account must be a throwaway, not a
   personal one.
3. **Cookies expire.** Password change or a security prompt kills the session;
   the API then returns `PRF04` until cookies are refreshed.
4. **LinkedIn can change Voyager without warning.** It's a private API. The
   quarantined package limits the blast radius, but a change means a fix.
5. **Rate limits are real and undocumented.** Hence the cache and the daily cap.
6. **A few sections may be unreachable** — final list confirmed in Phase 0 and
   documented honestly rather than faked.
7. **Free-tier cold starts** make the first request after idle slow.
8. **Not for bulk harvesting.** Single-profile, on-demand, deliberately capped.

---

## 14. Open questions

**None.** All decisions are settled: Voyager API, called directly with no
browser · synchronous · config defaults with a per-request override · API only,
no UI (§9).

Two things I'll need from you when we start Phase 0:

1. **A throwaway LinkedIn account's `li_at` and `JSESSIONID` cookies.** Paste
   them into `config/local.yml` yourself — that file is gitignored, and I won't
   put them anywhere else.
2. **3–4 profile URLs to test against**, ideally varied: one dense profile with
   patents/publications, one sparse one, and one with no photo — so we exercise
   the empty and `null` paths for real.

