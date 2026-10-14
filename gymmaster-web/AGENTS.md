<!-- BEGIN:nextjs-agent-rules -->

## This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

## GymMaster conventions

- Read `README.md` (structure, auth flow, module checklist) and `DESIGN_SYSTEM.md` before changing UI.
- UI is Vietnamese only: every string goes in `messages/vi/<domain>.json` — no hardcoded text in components.
- Data fetching: `apiClient` (`src/lib/http.ts`) + TanStack Query hooks in `features/<module>/queries.ts`. No axios/SWR.
- Icons: `@heroicons/react` only. No arbitrary Tailwind values outside `src/components/ui/`.
- API contract lives in `../docs/api/`. Verify with `npm run typecheck && npm run lint`.
