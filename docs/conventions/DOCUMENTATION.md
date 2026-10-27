# Documentation standard

Keep one canonical guide per topic. `AGENTS.md` files and package READMEs route readers to it instead of copying it.

## Place and name

- Choose the topic folder from the [documentation home](../README.md). Use UPPERCASE Markdown file names with the `.md` extension, such as `LOCAL-SETUP.md`. Each index is `README.md`. Folder names stay lowercase.
- Keep human guidance in `docs/`. Fixtures, executable contracts, environment templates, and runtime data stay beside their consumers.
- Root/package READMEs remain short entry points. Keep asset provenance beside the assets; do not create parallel package documentation trees.
- Every real folder under `docs/` has a `README.md` listing its documents and child folders. Update indexes in the same change and list only real entries.
- Keep the tree flat until a topic has companion files. Do not create empty categories, duplicate role manuals, or unused templates.

## Write and link

- State the purpose or outcome first. Use English, concrete names, short paragraphs, and commands that match the repository.
- Use relative Markdown links to existing repository files. Living guides avoid pinned line numbers; preserve compatibility anchors when content moves.
- Give each rule one home and link to it. Tool settings and agent entry points must not become duplicate manuals.
- Label plans as intent, historical records with their date/context, and operational examples with prerequisites. An old successful run is not a current result.
- Describe validation precisely: builds, authenticated workflows, deployments, and physical-device checks establish different things.

## Change safely

Before a rename or deletion, use `rg` to find inbound references throughout the repository. Update Markdown/HTML links, scoped guides, indexes, and retained anchors together.

Verify local targets and protected content after editing. During this modular refactor, keep the root Product tour section and original architecture SVG byte-identical; reorganizing prose does not authorize changing them.
