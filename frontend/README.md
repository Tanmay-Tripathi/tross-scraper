# tross-scraper frontend

Standalone Vite + React 19 + TypeScript SPA for the [tross-scraper](../README.md) API.

## Setup

```bash
cp .env.example .env
npm ci
npm run dev        # http://localhost:4204
```

## Scripts

| Command          | Purpose                                 |
|------------------|-----------------------------------------|
| `npm run dev`    | Dev server on :4204                     |
| `npm run build`  | Type-check, then build to `dist/`       |
| `npm run preview`| Serve the production build on :4204     |
| `npm run lint`   | Biome check with autofix                |
| `npm run format` | Biome format                            |

## Environment

| Variable          | Purpose                                                  |
|-------------------|----------------------------------------------------------|
| `VITE_APP_NAME`   | Name shown in the header                                 |
| `VITE_API_HOST`   | Base URL of the Go API; empty means same-origin          |
| `VITE_LOG_ERRORS` | `true` enables console logging from the error boundary   |

Vite **inlines these at build time**, so changing `VITE_API_HOST` requires a rebuild.

## Layout

```
src/
├── main.tsx              # React entry (BrowserRouter)
├── App.tsx               # route table
├── app/                  # layout, pages, globals.css
├── base/config.ts        # runtime config + apiUrl() helper
├── components/ui/        # shadcn/ui components
├── lib/utils.ts          # cn() class merger
├── types/routes.ts       # route constants
└── utils/                # error boundary, logger
```

### Adding a route

1. Create the page under `src/app/<feature>/page.tsx`.
2. Add its path to `src/types/routes.ts`.
3. Register a `<Route>` in `src/App.tsx`.

## Docker

```bash
docker build --build-arg VITE_API_HOST=https://api.example.com -t tross-scraper-frontend .
docker run -p 4204:8080 tross-scraper-frontend
```

Served by nginx on port 8080 with an SPA fallback and long-lived caching for hashed assets.
