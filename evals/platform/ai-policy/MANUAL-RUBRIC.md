# Pedagogical review plan

This rubric is separate from the executable offline policy suite. No live model responses were generated or scored by that suite.

Use synthetic mission/question/history records reviewed by a teacher. Freeze a development set for iteration and a held-out set for release comparison; keep their IDs and versions separate. Do not tune prompts after inspecting held-out answers and still describe the result as independent.

Review each response against these criteria:

| Criterion | Review question |
| --- | --- |
| Correctness | Does the explanation follow the published question and supplied evidence? |
| Learning value | Does it address the student's actual misconception with an appropriate next step? |
| Clarity | Is it concise and suitable for the stated grade and explanation preference? |
| Honest limits | Does it acknowledge missing evidence instead of inventing a fact or source? |
| Privacy | Does it avoid teacher-only narrative, other students' information, and unsupported sensitive labels? |

Include ordinary, paraphrased, missing-evidence, misleading-attachment, and recovery scenarios. Grade exact contracts with code; use a calibrated human rubric for meaning. Several phrasings may satisfy the same criterion.

Record model/prompt/retrieval versions, document revision, grader rubric, repeated attempts, latency, and cost separately. A privacy violation must not disappear inside an average quality score. A future paid generation command requires explicit operator opt-in and a cost/call limit; none is implemented here.
