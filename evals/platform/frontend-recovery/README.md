# Frontend recovery policy evaluation

Run from the repository root with Python 3.9+ and Node.js 24+:

```sh
python3 evals/run.py --dataset evals/platform/frontend-recovery/cases.json
```

The Node bridge imports the production reducer, tutor policy, memory repair, and server checkpoint mapper. Python owns the synthetic inputs, assertions, and immutable reports. No rule is copied, no browser storage is used, and no provider or database client is created.

The thirteen cases cover all four explanation feedback choices, retrying a wrong answer before completion, preserving the incorrect evidence, repairing older local progress, revisiting an unfinished stage, resuming an already credited answer without double credit, mapping published question IDs, and avoiding duplicate retry queues, and nullable JSON checkpoints. Historical server completion remains unchanged.

These checks cover local policy. Use the [real browser store](../browser-store/README.md) and [local app flows](../local-http/README.md) to check IndexedDB, React recovery, Firebase rules, and actual UI behavior.
