# Mia prompt comparison rules

Use the same four synthetic questions for both versions. Read the message, all steps and the follow-up. Score each answer before looking at prompt labels or token counts.

| Check | Pass when |
| --- | --- |
| Task | The response meets the saved learning goal. For the missing-temperature question it must acknowledge the gap and avoid giving a number. |
| Source support | Every factual detail follows from the retrieved sentences, the expected answer if supplied, or the published mission title and goal. Accept paraphrases and direct implications. An example adds a claim if it introduces a location, cause or condition absent from that evidence. Check assumptions in questions too. |
| Follow-up | The question gives the learner a specific next step they can complete with the lesson alone. |
| Source IDs | The set returned by the API matches the case's expected IDs. |

Report the four scores separately. Count a fallback as a failure to deliver an accepted model answer. Prefer the shorter prompt on this sample if it improves source support, keeps task completion and follow-up scores at least level, and reduces measured input tokens. Also report total tokens, cost and latency, including trade-offs.

The eight saved answers are calibration examples. Their reference labels come from Codex. The fresh generations use the same questions, so this is a development comparison. Teacher review and a new set of questions would provide independent validation.

The transfer set uses two new passages on runoff and infiltration. Its four questions were written after the compact prompt was fixed and before inspecting fresh tutor answers. Run each once per version. Report this small transfer check separately from the repeated development set.

The criteria follow MindStudio's suggestion to define observable checks and to review failures one dimension at a time. [MindStudio guide to writing evals](https://www.mindstudio.ai/blog/how-to-write-evals-for-ai-agents)
