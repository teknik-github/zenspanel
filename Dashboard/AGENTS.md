# Repository Guidelines

## Project Structure & Module Organization

This is a Nuxt 4 dashboard application using Vue 3, Nuxt UI, Tailwind CSS, and pnpm. Application code lives in `app/`: route views are in `app/pages`, shared UI in `app/components`, layouts in `app/layouts`, route middleware in `app/middleware`, composables in `app/composables`, types in `app/types`, utilities in `app/utils`, and global CSS in `app/assets/css/main.css`. Static browser assets belong in `public/`. Server-side Nuxt API handlers live in `server/api`; `server/api/v1/[...path].ts` proxies dashboard API calls to the backend configured by `runtimeConfig.backendUrl`.

## Build, Test, and Development Commands

Use pnpm 11, matching `packageManager` in `package.json`.

- `pnpm install` installs dependencies and runs `nuxt prepare`.
- `pnpm dev` starts the local Nuxt development server.
- `pnpm build` creates a production build.
- `pnpm preview` serves the production build locally.
- `pnpm lint` runs ESLint across the repository.
- `pnpm typecheck` runs Nuxt/Vue TypeScript checks.

## Coding Style & Naming Conventions

Write Vue single-file components with `<script setup lang="ts">` where practical. Keep route files aligned with Nuxt file routing, for example `app/pages/users/index.vue` for `/users`. Name reusable components in PascalCase, composables as `useThing.ts`, and shared types with clear domain names under `app/types`. Follow the Nuxt ESLint config in `eslint.config.mjs`: no trailing comma dangles, 1TBS braces, and at most three Vue attributes on a single line before wrapping.

## Testing Guidelines

No dedicated test framework or `pnpm test` script is currently configured. For changes, run `pnpm lint`, `pnpm typecheck`, and `pnpm build` before opening a PR. If tests are added, prefer colocated `*.spec.ts` files or a top-level `tests/` directory, and document the new command in `package.json` and this guide.

## Commit & Pull Request Guidelines

Recent history uses conventional, imperative prefixes such as `fix:`, `docs:`, `refactor:`, and `security:`. Keep commits scoped and specific, for example `fix: package update missing limits`. Pull requests should include a short summary, verification commands run, linked issue or task context, and screenshots for visible UI changes.

## Security & Configuration Tips

Do not commit secrets. Use `.env.example` as the template for local configuration. Keep backend access server-side through Nuxt runtime config rather than exposing it publicly. For dependency vulnerability checks, run `cve-lite . --json` from the project root and review the generated `cve-lite-scan-<timestamp>.json` before applying suggested fix commands.
