# Frontend recovery policy evaluation

Run from the repository root with Python 3.9+ and Node.js 24+:

```sh
python3 evals/run.py --dataset evals/platform/frontend-recovery/cases.json
```

The Node bridge imports the production reducer, tutor policy, memory repair, and server checkpoint mapper. Python owns the synthetic inputs, assertions, and immutable reports. No rule is copied, no browser storage is used, and no provider or database client is created.

The thirteen cases cover all four explanation feedback choices, continuing after reviewing a wrong answer without correct-answer credit, preserving the incorrect evidence, repairing older local progress, revisiting an unfinished stage, resuming an already credited answer without double credit, mapping published question IDs, and removing reviewed questions from duplicate pools, and nullable JSON checkpoints. Historical server completion remains unchanged.

These checks cover local policy. Use the [real browser store](../browser-store/README.md) and [local app flows](../local-http/README.md) to check IndexedDB, React recovery, Firebase rules, and actual UI behavior.

## Policy change in v3

The requested product behavior now lets a student move on after reviewing an incorrect answer. Six existing scenarios were revised; no scenario was added. Correct-answer counts and incorrect evidence remain checked. The following v1 IDs are retired; their original definitions remain in [the previous revision](https://github.com/velrino/classgame/blob/e1b255691b8e5b7ffa8bf59b505f5cf901d746bd/evals/platform/frontend-recovery/cases.json).

| Retired ID | Replacement ID |
| --- | --- |
| `wrong-helpful-must-retry-v1` | `wrong-helpful-review-and-continue-v2` |
| `wrong-confusing-must-retry-v1` | `wrong-confusing-review-and-continue-v2` |
| `wrong-too-easy-must-retry-v1` | `wrong-too-easy-review-and-continue-v2` |
| `wrong-too-hard-must-retry-v1` | `wrong-too-hard-review-and-continue-v2` |
| `retry-correct-advances-v1` | `wrong-then-next-correct-advances-v2` |
| `retry-pool-does-not-repeat-current-v1` | `reviewed-pool-does-not-repeat-current-v2` |
