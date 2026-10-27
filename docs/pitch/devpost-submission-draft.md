# Devpost Submission Draft

> Historical submission material. Former hosted links and deployment descriptions are preserved as context; use [local setup](../development/local-setup.md) and [hosting history](../operations/hosting-history.md) for current instructions.

This file is a draft for the **All Things Agentic Hackathon** Devpost form. Last synced with the code on **August 29, 2026**.

Hackathon page: https://allthingsagentichackathon.devpost.com/

Category brief (The Collaborative Partner — in depth):

> **The focus:** Stateful, multi-turn dialogue with real-time context retrieval (RAG) and persistent memory, so your agent adapts and personalizes based on past interactions instead of starting over each time.
>
> **Examples:** An expert guide that helps you understand a dense legal document, quizzes you as you go, learns which concepts you struggle with, and adapts future explanations. A UI/UX helper for non-designers that turns a vague idea into a wireframe and learns your brand preferences from your corrections.

## Project Overview

### Project Name

ClassGame

### Elevator Pitch

Supporting teachers as they guide their class through Mia, an AI tutor in a 3D world of missions that adapt to every student — rewarding the thinking, not just the right answer.

## Project Details

### About The Project

## Inspiration

I have spent years building tools for teachers, and one complaint never goes away: homework tells you _who_ got it wrong, never _why_. A stack of wrong answers is a pile of question marks the teacher has to guess through the next morning.

Most educational apps make it worse. Every student gets the same quiz and the same next step, the app never talks back to the teacher, and every session starts from zero. The good tutor at the kitchen table does the opposite: she asks why you picked that, notices what keeps tripping you up, changes how she explains, and remembers you tomorrow.

The Collaborative Partner brief describes exactly that tutor. ClassGame puts her in a 3D classroom on a student's phone, gives her a memory that survives the tab and the mission, and — the part I cared about most — makes what she learns about each student useful to the teacher.

**ClassGame is the optional homework that takes notes for the teacher.**

## What it does

**The teacher** signs in, creates a class, writes what the lesson covered and what she noticed, and optionally attaches the real worksheets (PDF, images, text). **Mia** — the Multimodal Intelligence Agent, powered by Gemini — turns that into a mission: 3–4 stages, questions whose wrong options are deliberate traps tied to _named_ misconceptions, step-by-step explanations, a lesson flow map, and a set of knowledge chunks distilled from the teacher's material that Mia will retrieve from while tutoring. The teacher reviews the preview, adjusts the prompt, regenerates (every revision stays on the same mission), and approves. The mission appears in the class in real time.

**The student** enters the class's six-digit code and a name, picks a character, and walks into a low-poly 3D classroom. A glowing circle appears by the desk the moment the teacher publishes. Mia runs the mission on the chalkboard:

- She asks every question and decides every next step.
- When the answer is wrong she does not reveal the solution. She asks _why you picked it_, with reasons written in the student's voice and generated for that specific wrong answer.
- She explains in short steps or draws it, depending on what has worked for this student before.
- The student rates each explanation — helpful, confusing, too easy, too hard — and Mia adapts and says why: _"You marked 'confusing' → I'll draw the next one."_
- Close the tab, come back tomorrow: _"Welcome back — last time you were on Fractions."_ Same question, same step, same lines on the board.

**Back on the dashboard**, the teacher watches the class live from Firestore: who has not started, who is at the desk with Mia right now, who left early, who finished. Then what Mia observed per student: a summary, strengths, areas to revisit, a learning-flow graph of the path they took, and the evidence behind every claim — each answer, each explicit reason, each feedback tap, each tutor turn with Mia's teacher-only observation and the ids of the chunks she retrieved. Across the class: clustered strengths and needs and a tiered plan for the next lesson. One button, **Build tomorrow's mission**, turns those patterns into the next prompt. That closes the loop.

## How we built it

One repository, two Cloud Run services, one database, built alone in three days.

**The agent.** Go with the Google GenAI SDK on Vertex AI (`gemini-3.7-flash`), structured JSON output, four generators in one orchestrator: `BuildMission`, `BuildTutorTurn`, `BuildLearningTrace` (after a completed _or abandoned_ attempt) and `BuildClassLearningSynthesis`. Every generator has a deterministic fallback, so no endpoint fails because of the model, and every output records provider, model, `generated_by_ai`, `fallback_reason` and token usage.

**One tutor turn.** The game dispatches a deterministic tutor locally so the chalkboard never waits; the API turn overwrites Mia's message, the "why" prompt, the explanation steps and the style/difficulty when it lands. Server-side: ground the request against the pinned published question → retrieve the top 5 knowledge chunks for _this question and this response_ → load the student's memory, the attempt history (last 24 observable events, last 12 tutor turns) and the class context → Gemini → a **leak guard** that discards any output mentioning teacher-only or class context → persist idempotently by event id → return only student-safe fields.

**Memory.** Three layers in Firestore: the **attempt** (progress, events, decisions, exact checkpoint, tutor turns, trace), the **student** (`learning_memory`: strengths, improvement areas, preferred explanation style, `next_adaptation` — carried into the next mission) and the **class** (clustered patterns across students). The game writes a monotonic snapshot 450 ms after every state change, every 30 s while visible, and on `pagehide`. Cloud Tasks runs finalization on completion and a 120 s abandonment watchdog that still writes a partial trace for a student who gave up; the same queue copies attachments into Cloud Storage. Redis rate-limits generation per teacher, student and IP. Firebase Auth: Google for teachers, anonymous for students.

**Frontend.** Next.js 16, React 19, TypeScript, Tailwind, Three.js. The classroom is one merged GLB split into connected islands at load time to stretch the room, seat classmates and derive colliders; the six chibi characters are repainted per instance by rewriting vertex colours. Avatars are generated headless in Blender (one shared rig, Draco), which I drove through its MCP server from Claude Code to inspect and iterate. The dashboard uses Firestore realtime listeners, React Flow for the mission and learning graphs, Uploadcare for uploads, PostHog for product and error analytics.

**Safety and tenancy.** Firestore rules are read-only for browsers — teachers see only their own classes, students only a student-safe projection. Mission generation has a deterministic pre-model gate and a post-model education guardrail; blocked missions cannot be published. Both tutor and trace prompts reason only from observable answers and are forbidden from claiming to read a student's mind. Every durable record already carries `organization_id`, `district_id` and `school_id`.

## Challenges we ran into

**Keeping the student's screen honest.** The first version showed Mia's notebook to the student. It felt agentic and it was wrong: a 12-year-old should not read "confuses numerator and denominator" about themselves. I moved every observation to the teacher side, made the API return a student-safe projection, denied anonymous reads in Firestore rules, and added the leak guard.

**Never starting over — without faking it.** "Persistent memory" is easy to fake with `localStorage`. Real persistence meant revision numbers and server-side merge with optimistic concurrency, a checkpoint that includes the lines on the board and the pending decision, revision pinning so a republish cannot swap questions under a running attempt, and a watchdog that finalizes what a student left behind.

**Fast and grounded at the same time.** A model call per tap makes a chalkboard feel slow. The local deterministic tutor plus the overwriting structured turn keeps the game responsive, and grounding on the server — never the client — closed the door on a browser inventing "expected answers".

**Structured output under pressure.** Gemini's structured-output endpoint rejected schema keywords I took for granted (`maxItems`, for one), so a generator that worked in the morning failed after a schema edit in the afternoon. I added a sanitizer that strips unsupported keywords recursively before every call, normalize every blueprint after it, and keep the deterministic fallbacks identical in shape — the UI never sees the difference.

**Furnishing a room from one mesh.** Seating a classmate in every chair meant recognising chairs and desk tops by their dimensions and measuring the rig's leg pivot. Repainting characters hit 8-bit colour quantisation — near-black hair and near-black shoes became the same colour — solved by matching vertices only against parts that can exist at their height.

## Accomplishments that we're proud of

- **Both collaboration loops run live end to end**: teacher → mission → student → evidence → next mission, with no manual step in between and no mocked data in the demo.
- **Mia explains every decision.** Difficulty, explanation style and next action all carry a stated reason the teacher can read.
- **Retrieval you can audit.** Every tutor turn stores the chunk ids it used, and the dashboard shows them next to the reply.
- **Memory that resumes exactly.** A student who closes the tab mid-question returns to the same phase and step, and a student who never returns still produces a partial learning trace.
- **A student never sees a note about themselves**, enforced in three places: rules, API projection, and the model's own output gate.
- **125 commits in three days**, solo, shipping a Go API, a Three.js game, a realtime dashboard and a 3D asset pipeline to production on Cloud Run.

## What we learned

The strongest tutoring agent is not the one that generates the most content; it is the one that turns small signals — a chosen reason, a "confusing" tap, a repeated slip, a tab closed mid-question — into a decision it can explain and a memory it can act on later.

Retrieval did not need embeddings to be useful. Chunks distilled from the teacher's own material, retrieved lexically and shown to the teacher by id, earned more trust than a black-box vector search would have in a classroom.

Provenance matters as much as the answer. Labelling every block as Gemini, fallback or example is what lets a teacher — or a judge — trust what they see.

And, on a personal note: with a 3D world eating every hour, the discipline that saved the project was asking of each feature, "does this make the agent's behaviour more visible?" and cutting everything that did not.

## What's next for ClassGame

I did not build ClassGame to win a hackathon. I built it because I have watched teachers spend their evenings guessing what went wrong in thirty heads, and I believe that time belongs to their families and their students, not to a pile of paper. Winning would let me stop treating this as a three-day sprint and start treating it as the thing I actually want to spend the next years on: making a teacher's life lighter and a child's learning better, at the same time, with the same product.

Here is the dream, in the order I want to chase it.

**More worlds, more subjects.** Today Mia teaches Grade 5 math on one chalkboard. I want her in a science lab where a wrong hypothesis makes the beaker fizz, in a history classroom where the map redraws itself as the student argues, in a language room where the conversation partner is another chibi. Every scenario is a new set of misconceptions to model — and a new way for a kid to _want_ to do the optional homework.

**Play that teaches.** The 3D room is still mostly a stage. I want the game itself to carry the pedagogy: the "pick the next move" equation duels I prototyped, cooperative missions where two classmates have to explain to each other before Mia lets them advance, a classroom that visibly changes as the class masters something together. Fun is not decoration here; it is the reason a student comes back on a Saturday.

**Rewards that follow the thinking.** Today a student earns the next stage and a progress bar that fills — honest, but thin. I want the loop to pay off: points for explaining a mistake rather than for guessing right, streaks for coming back on their own, characters and classroom items unlocked by mastering a topic instead of by grinding, and a class goal the group only reaches together. One rule holds it in place: the reward follows the reasoning. A student who answers wrong, says why, and gets there should out-earn one who guessed correctly — otherwise the game teaches the opposite of what Mia does.

**A tutor that gets wiser.** Semantic retrieval over the teacher's material next to the traceable lexical pass, memory that spans weeks and subjects instead of one mission, and Mia's voice on the chalkboard so a student who struggles with reading is not locked out of the help.

**Scale without losing the notebook.** The tenancy is already in every record — organization, district, school. I want a district to see, across hundreds of classrooms, which misconceptions travel together, and a new teacher to open ClassGame on day one and inherit what Mia already knows about her students. All of it with the same rule that shaped this project: the student never reads a note about themselves, and the teacher always sees the evidence.

**Real classrooms.** First pilots with the teachers whose complaint started this. Their corrections, not my roadmap, decide what ships next.

If ClassGame gets that chance, the win is not the prize. It is a teacher opening the dashboard on Monday morning, already knowing why — and a student walking up to a glowing desk because they want to.

## Built With

Suggested Devpost tags:

- Gemini
- Gemini 3.7 Flash
- Google GenAI SDK
- Vertex AI
- Google Cloud
- Cloud Run
- Firestore
- Cloud Tasks
- Cloud Storage
- Firebase Auth
- Cloud Build
- Go
- Next.js
- React
- TypeScript
- Three.js
- React Flow
- Redis
- Blender
- Blender MCP
- Model Context Protocol (MCP)
- Node.js
- Education
- AI Agents
- Generative AI
- RAG
- Game-Based Learning
- Adaptive Learning
- Personalized Learning
- WebGL
- Serverless

## Try It Out Links

### Hosted Project URL

```txt
https://classgame-web-mzkj6f7g3a-ue.a.run.app/home
```

Teacher side: `https://classgame-web-mzkj6f7g3a-ue.a.run.app/login` (demo account pre-filled).

### Code Repository

```txt
https://github.com/velrino/classgame
```

### Demo Video Link

TODO: Add public YouTube or Vimeo demo video URL (under 4 minutes, English or English subtitles, shows Cloud Run).

```txt
https://www.youtube.com/watch?v=<video-id>
```

## Additional Info For Judges

### Sponsor / Special Prizes

Do not select **Startup Excellence** unless submitting through an incorporated organization with a corporate email address.

```txt
No Startup Excellence opt-in.
```

### Submitter Type

```txt
Individual/Hobbyist
```

### Submitter Country Of Residence

TODO: Confirm before submission.

```txt
Brazil
```

### Category

```txt
The Collaborative Partner
```

### Organization Name

```txt
N/A
```

### Project Start Date

First commit in the repository: August 27, 2026.

```txt
08-27-26
```

### Public Or Private Code Repo URL

```txt
https://github.com/velrino/classgame
```

If private, share access with:

```txt
testing@devpost.com
cloudhackathons@google.com
```

### Reproducible Testing Instructions Added To README

```txt
Yes
```

[local setup](../development/local-setup.md) → _Run it locally_ and _Testing instructions (hosted)_.

### Hosted Project URL If Available

```txt
https://classgame-web-mzkj6f7g3a-ue.a.run.app/home
```

### Testing Instructions

Smoke tests on the deployed services:

```bash
curl -I https://classgame-web-mzkj6f7g3a-ue.a.run.app/home
curl https://classgame-api-2057331207.us-east1.run.app/
curl -X POST https://classgame-api-2057331207.us-east1.run.app/users/student-session/validate \
  -H 'Content-Type: application/json' --data '{"class_code":"000000"}'   # 404 class_not_found
```

Local run:

```bash
cd apps/api && cp .env.example .env && go run ./cmd/api      # :4004, needs Firebase creds + Redis; GENAI_ENABLED=false works with fallbacks
cd apps/web && cp .env.example .env.local && npm install && npm run dev   # :3000
```

Form text:

```txt
No paid account is needed. Two browser windows (or a laptop and a phone) work best.

Teacher:
1. Open https://classgame-web-mzkj6f7g3a-ue.a.run.app/login. The demo teacher account is pre-filled — click "Sign in" (or use any Google account).
2. Create a class (name, subject, grade) or open an existing one.
3. In the Missions tab, enter a mission name, a learning goal and a teacher prompt, e.g. "Today: solving two-step equations. Several students slipped on fractions." Optionally attach a worksheet PDF (../../assets/grade/README.md has ready-to-copy prompts for the included Grade 5 worksheets). Click "Generate preview".
4. Review the preview: Gemini generates stages, questions with misconception traps, the mission flow graph and an education guardrail. Edit the prompt and click "Update preview" if you like — revisions stay on the same mission.
5. Click "Approve and assign to class" and keep the class's six-digit code visible.

Student (a phone works best):
6. Open https://classgame-web-mzkj6f7g3a-ue.a.run.app/home, enter the class code and a name, pick a character, walk to the glowing desk, tap "START", and choose the published mission.
7. Answer the first question wrong on purpose. Mia asks why — pick a reason. Read the explanation, then tap "Confusing": Mia switches to a drawing and the decision card states why.
8. Answer a few more questions. Close the tab and reopen the same URL in the same browser: "Welcome back — last time you were on …", and you are back at the exact question and step.
9. Finish the mission (or walk away for two minutes: the watchdog marks the attempt as left early and still writes a partial learning trace).

Teacher again:
10. The class Overview updates live: the student moves through the mission funnel, appears in Recent players with online state, and the class strengths / areas for growth refresh when the learning trace lands.
11. Open the student in the Students tab: Summary (Gemini's summary, strengths, areas to revisit), Learning flow (graph of the path), Evidence (every event and every tutor turn — Mia's reply, the teacher-only observation and the retrieved chunk ids), Memory (the next adaptive turn and provenance).
12. Click "Build tomorrow's mission": the class patterns are pre-written into the next prompt.

Teacher sign-in uses Google or the pre-filled demo account; students sign in anonymously.
```

### Which Google SDK Did You Use?

```txt
Google GenAI SDK (google-genai)
```

Go module `google.golang.org/genai`, Vertex AI backend.

### Which Google Cloud Services Did You Use?

```txt
Cloud Run
Firestore
Cloud Tasks
Cloud Storage
Vertex AI
Cloud Build
```

Firebase Auth is used for teacher (Google / password demo account) and student (anonymous) sign-in. Do not select Pub/Sub, Cloud SQL, GKE, Cloud Scheduler or Memorystore — not used.

### Architecture Diagram

TODO: Upload `architecture-diagram.png` or `architecture-diagram.pdf`.

Suggested diagram contents:

```txt
Teacher dashboard (Next.js on Cloud Run) — /login, /dashboard, /dashboard/class/{classId}
  -> Firebase Auth (Google Identity Services / demo account)
  -> Uploadcare temporary upload for optional files
  -> Go API on Cloud Run
       POST /teacher-classes
       POST /teacher-classes/{class_id}/missions/preview
         -> Redis rate limit
         -> Google GenAI SDK -> Gemini 3.7 Flash structured output (attachments inline)
         -> Firestore missions/{mission_id} (tenant-scoped draft revisions, knowledge_chunks)
       POST /teacher-classes/{class_id}/missions/{mission_id}/approve
         -> Firestore missions/{mission_id}/published_revisions/{n}
         -> student_class_missions/{class_id}/missions/{mission_id} (student-safe realtime metadata)
         -> Cloud Tasks -> Cloud Storage (attachments per class / mission / revision)
       GET  /dashboard/overview?class_id=…
  -> Firestore realtime listeners (student_mission_attempts, class_learning_summaries)

Student game (Three.js, Next.js on Cloud Run) — /home
  -> POST /users/student-session/validate
  -> Firebase Auth (anonymous) -> POST /users/student-session -> users/{uid} + class lobby
  -> Firestore onSnapshot (class, published mission catalog, own completions)
  -> POST /users/student-mission -> selected pinned mission payload
  -> POST /users/student-presence every 15 s
  -> POST /users/student-tutor-turn per answer / why / feedback
       -> grounding + retrieval (knowledge chunks, student memory, attempt history, class context)
       -> Gemini 3.7 Flash structured output -> leak guard
       -> Firestore student_mission_attempts/{attempt_id}.tutor_turns[]
  -> POST /users/student-learning (snapshot + exact checkpoint)
       -> Cloud Tasks finalize / 120 s watchdog
       -> Gemini learning trace -> users/{uid}.student.learning_memory
       -> Gemini class synthesis -> class_learning_summaries/{class_id}
  -> GET /users/student-learning?mission_id=… (exact resume)
```

### Pre-existing Or Third-party Code And Assets

Disclose on the form:

```txt
- Low Poly ClassRoom 3D model by Mumladze28 (Sketchfab), adapted at load time.
- Chibi avatar set generated for the author's earlier the author's prior project project (base mesh, hair and cap licensing in ../../assets/3d/avatars/chibi-band/README.md).
- Grade 5 worksheets from K5 Learning (free printables) used as test material for mission generation.
- shadcn/ui components; the dashboard shell layout was adapted from the open-source shadcn-admin template.
- Everything else — game, tutor loop, API, orchestrator, memory model, dashboard — was written during the hackathon.
```

### Startup Prize Organization Name

```txt
N/A
```

### Startup Prize Corporate Email

```txt
N/A
```

### Which Google AI Models Did You Use?

```txt
Gemini 3.7 Flash
```

Production sets `GENAI_MODEL=gemini-3.7-flash` in `api/infra/cloud-run/production/cloudbuild.yaml`. Only add Gemma, Veo or Lyria if actually integrated.

```txt
Optional bonus models: None
```

### Optional Bonus Content Link

TODO: Add public blog/video link if created. The content must include:

```txt
I created this piece of content for the purpose of entering the All Things Agentic Hackathon.
```

### Optional Social Media Post

TODO: Add public social media post link if created.

Suggested post:

```txt
Building ClassGame for the #AllThingsAgenticHackathon: the optional homework that takes notes for the teacher. Mia, the Multimodal Intelligence Agent powered by Gemini, turns today's lesson into a 3D review mission, asks students why they got things wrong, remembers how each one thinks across sessions, and hands the teacher the evidence — live. Google GenAI SDK + Cloud Run + Firestore + Cloud Tasks.
```

## Final Submission Checklist

- [ ] Joined the hackathon on Devpost.
- [ ] Requested Google Cloud credits.
- [x] Built the project during the submission period (first commit August 27, 2026).
- [x] Used Gemini 3.5 or newer (`gemini-3.7-flash`).
- [x] Used Google GenAI SDK.
- [x] Used Cloud Run, Firestore, Cloud Tasks and Cloud Storage.
- [ ] Selected **The Collaborative Partner** category.
- [x] Public code repository.
- [x] Reproducible setup and testing instructions in [local setup](../development/local-setup.md).
- [ ] Uploaded architecture diagram.
- [ ] Uploaded public YouTube/Vimeo video under 4 minutes.
- [ ] Video is in English or has English subtitles.
- [ ] Video shows Cloud Run / Google Cloud proof.
- [x] Hosted demo URL works.
- [ ] Any private repo is shared with `testing@devpost.com` and `cloudhackathons@google.com` (not needed if public).
- [ ] Pre-existing and third-party code and assets disclosed (text above).
- [ ] Repo, video and linked materials are not changed after the deadline until winners are announced.
