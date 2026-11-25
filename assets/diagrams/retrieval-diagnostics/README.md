# Retrieval diagnostic diagram

`diagram.png` explains the separate synthetic live sample in PR #126. The two columns compare versions; they are not one runtime flow. Before: no source and zero model calls. After: source recovered and one real Gemini call. The visible human-review note states that extra answer details were absent from the source. This is not a general accuracy result.

Generated with the built-in ImageGen tool on 2026-11-15. Selected master: `exec-2282ea87-fa57-4e3c-b1ce-6948bf8f0375.png`. The existing blog comparison supplied a style reference only; its offline model-disabled wording was replaced with the measured live-sample facts. White background, thin navy lines and cyan/violet panels remain readable in light and dark GitHub themes. This is separate from the system architecture and numbered mission-flow diagrams.

Art direction: landscape 3:2; title “Same question. Better evidence.”; query “Explain ‘condensation’” with curly double quotes. Before: quotes hide matching word → no lesson fact → safe fallback → zero calls. After: quotes become separators → fact recovered → Gemini explains phase change → one call. Source strip: water vapor changes into liquid droplets. Review strip: extra details were not in the source. No statistics or benchmark claim.

Checked English labels, downward arrows, call counts and the review limitation after generation. Normal builds use the checked-in PNG and need no generation tool.

See [method and observations](../../../docs/ai/RAG-RETRIEVAL-DIAGNOSTICS.md) for the full evidence and separate offline comparison table.
