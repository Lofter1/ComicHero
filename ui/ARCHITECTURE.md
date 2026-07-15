# Frontend architecture

The frontend uses Vue 3 with the Composition API. Code is organized by ownership rather than by file type so that a feature can usually be understood from one directory.

## Directory map

```text
src/
├── app/                 # Application-wide shell components
├── api/                 # HTTP transport and endpoint functions
├── features/            # User-facing features and their private implementation
│   └── <feature>/
│       ├── components/  # Views and feature-specific UI
│       ├── model.js     # Pure data mapping and payload helpers, when needed
│       └── use*.js      # Feature state and API workflows
├── router/              # URL definitions, guards, and URL-to-view translation
├── shared/              # Reusable UI and behavior with no feature ownership
│   ├── components/
│   └── composables/
├── styles/              # Styles grouped by responsibility and breakpoint
├── App.vue              # Cross-feature composition only
└── main.js              # Vue bootstrap
```

## Where new code belongs

- Put a screen, editor, or workflow in its owning `features/<feature>` directory.
- Put API calls in `api/client.js`; feature composables call the API and own loading/error state.
- Put pure conversions between API data and forms/payloads in a feature `model.js`.
- Promote a component to `shared/` only when at least two features use the same UI contract.
- Keep route names and paths in `router/index.js`; keep translation between route objects and application view state in `router/appRouteState.js`.
- Keep `App.vue` focused on wiring features together. Feature-specific requests, confirmation text, and state transitions belong in the feature composable.

## Component contracts

- Props carry data down; events report user intent up.
- Shared components should not call feature APIs.
- Prefer named slots for the small parts of a repeated layout that vary. `BrowseEntityRow` and `BrowseListSection` are the reference pattern.
- Use `v-model` only when a component is explicitly editing caller-owned form state.
- Avoid creating a shared abstraction for a single caller; local, obvious code is easier to maintain than a generic component with many switches.

## State and API workflow

Feature composables expose refs plus action functions. They receive cross-feature dependencies explicitly, which keeps imports acyclic and makes ownership visible at the call site in `App.vue`.

Global browser concerns live in shared composables:

- `useTheme` owns system-theme observation and persistence.
- `useListOptions` owns persisted browse filter/sort options.
- `usePagination` owns paged-list loading state.

## Styling

`styles.css` is the ordered entry point. Each imported file has one documented responsibility. Responsive overrides are separated by breakpoint. When adding a style:

1. Prefer an existing semantic class over element-specific selectors.
2. Put the base rule in the responsibility module.
3. Put only the changed properties in the relevant responsive module.
4. Do not add page-specific overrides to a shared component unless every caller should receive them.

## Verification

Run these from `ui/` before committing:

```sh
npm run format
npm run lint
npm test
npm run build
```

There is currently no frontend unit-test command. Add tests alongside a feature when introducing logic that is difficult to verify through the production build alone.
