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
│   │   ├── browse/      # Search/filter/sort and browse-row building blocks
│   │   ├── content/     # Content status and progress presentation
│   │   ├── detail/      # Repeated detail-page structure
│   │   ├── feedback/    # Loading, empty, error, and status feedback
│   │   ├── form/        # Buttons and form controls
│   │   ├── layout/      # Headers and panel-level layout
│   │   └── overlay/     # Dialog and overlay framing
│   └── composables/
├── styles.css           # Tailwind entry point, theme tokens, and element defaults
├── App.vue              # Cross-feature composition only
└── main.js              # Vue bootstrap
```

## Where new code belongs

- Put a screen, editor, or workflow in its owning `features/<feature>` directory.
- Put API calls in `api/client.js`; feature composables call the API and own loading/error state.
- Put pure conversions between API data and forms/payloads in a feature `model.js`.
- Promote a component to `shared/` only when at least two features use the same UI contract. A reusable component with one current caller stays feature-local until a second owner appears.
- Keep route names and paths in `router/index.js`; keep translation between route objects and application view state in `router/appRouteState.js`.
- Keep `App.vue` focused on wiring features together. Feature-specific requests, confirmation text, and state transitions belong in the feature composable.

## Component contracts

- Props carry data down; events report user intent up.
- Shared components should not call feature APIs.
- Prefer named slots for the small parts of a repeated layout that vary. `BrowseEntityRow` and `BrowseListSection` are the reference pattern.
- Use `v-model` only when a component is explicitly editing caller-owned form state.
- Avoid creating a shared abstraction for a single caller; local, obvious code is easier to maintain than a generic component with many switches.
- Shared visual building blocks live under `shared/components`: form controls own their variants and sizes, feedback components own status presentation, layout components own repeated panel structure, and overlay components own modal framing. Feature-local components such as `UserAccessSettings` and `ReadingOrderEntryPagination` own the appearance of their workflow without expanding the shared API.

## State and API workflow

Feature composables expose refs plus action functions. They receive cross-feature dependencies explicitly, which keeps imports acyclic and makes ownership visible at the call site in `App.vue`.

Global browser concerns live in shared composables:

- `useTheme` owns system-theme observation and persistence.
- `useListOptions` owns persisted browse filter/sort options.
- `usePagination` owns paged-list loading state.

## Styling

`styles.css` is the Tailwind entry point and owns theme tokens plus application-wide element defaults. Component appearance stays with the component. When adding a style:

1. Use short, semantic classes in the template.
2. Put their Tailwind utilities in a scoped style block with `@reference` and `@apply`.
3. Add a prop or variant to a shared component when every caller should receive the appearance; parent classes should control only layout and placement.
4. Keep responsive changes beside the component. Use a container query when a component must respond to its own available width, and a media query when the whole viewport drives the change.

The shared visual primitives currently include `BaseButton`, `BaseSelect`, `BaseTextInput`, `DetailPanel`, `MetadataGrid`, `PanelHeader`, `EmptyState`, `ProgressBar`, `StatusPill`, and `ModalShell`.

## Verification

Run these from `ui/` before committing:

```sh
npm run format
npm run lint
npm test
npm run build
```

The test command covers the current pure routing, reading-order model, and click-outside behavior. Add focused tests alongside a feature when introducing logic that is difficult to verify through the production build alone.
