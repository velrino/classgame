# ClassGame — 4-minute demo pitch

> Historical submission material. Former hosted links and deployment descriptions are preserved as context; use [local setup](../development/local-setup.md) and [hosting history](../operations/hosting-history.md) for current instructions.

Target runtime: **3:50–4:00** at a calm pace (~440 spoken words ≈ 110 words/min). Keep the Loom camera bubble visible and use the screen actions in brackets as silent cues.

**Names:** **Mia** — the **Multimodal Intelligence Agent** — is the AI agent (the UI says "Mia's preview", "Mia remembers…"). The human teacher and her in-game avatar remain **Emily**. Never call the AI Emily.

## Published video transcript

Source: the `0830.mp4` CapCut export posted to YouTube. Runtime: **3:53**.

This is a lightly cleaned transcript of the recorded video: obvious speech-to-text artifacts and project vocabulary were corrected (`ClassGame`, `Mia`, `Gemini`, `Firestore`, `Cloud Storage`, `Redis`, `Cloud Tasks`, `PostHog`).

**0:00–0:08**

Hi, my name is Denis, and I am here because I believe teachers and AI can work together to help students learn better.

**0:08–0:10**

And have fun doing it.

**0:10–0:19**

I have worked in tech for 12 years, and I built ClassGame alone for this hackathon, for desktop and mobile.

**0:19–0:26**

On the left, Bruce joins the game using the code from the teacher dashboard on the right.

**0:26–0:34**

But how does it work? For example, meet Emily, a fifth-grade math teacher. She wants her students to review what they learned in class today.

**0:34–0:43**

Now Emily uploads a PDF. Mia sends it to Gemini, and Gemini suggests three questions for Emily to approve.

**0:43–1:00**

In Preview, Emily checks every question and answer. And in Flow, she sees how Mia evaluates her students' answers. She can edit everything before approving.

**1:00–1:16**

And now, after Emily approves the mission, a glowing circle appears in Bruce's game. He walks to it, presses Start, and selects the new mission.

**1:16–1:26**

The student gets the first question right. We send the answer to Gemini while the student moves to the next question.

**1:26–1:48**

And now the student makes a mistake. Mia does not show the answer. Instead, she asks why. What happened? The student can choose a reason or write their own answer.

**1:48–2:04**

The student replies to the last question correctly. The mission is completed, and now the student receives an achievement for completing this mission.

**2:04–2:18**

Meanwhile, Mia analyzes the student's reasons to understand the mistake and how this student thinks. This gives teachers deep insight into each student.

**2:18–2:42**

And here, we can see the student's complete learning flow. Mia remembers this and uses it to provide better support in future missions for this student. And now, in the Overview, the teacher Emily can see insights about her class. This is an overview about everything that was processed.

**2:48–2:55**

And here is my favorite content, because it is about the architecture behind my game.

**2:55–3:14**

I prefer Next.js because I have a lot of experience with it. And I chose Go because it is very good. I really like it. It processes many tasks simultaneously, and Go is very good for this.

**3:14–3:21**

I created three agents to talk with Gemini and process simultaneously.

**3:21–3:31**

And I chose Firestore as a database because it is real-time and very good for this game.

**3:31–3:39**

And I chose Google Cloud Storage to store the teacher files.

**3:39–3:53**

And Redis as a database for cache and quotas too. And Cloud Tasks to process queues, and PostHog for analytics and exceptions.

Before recording:

- Prepare a Grade 5 math class with existing learning insights and open its overview. Rename the demo class from `TEST` to something like `Grade 5 Math` so it reads well on camera — every screenshot below still shows the old `Test` name.
- Open the student home and teacher dashboard in separate tabs.
- Start with the student join modal on the left and the teacher dashboard on the right, matching `../../assets/pitch/01-1.png`. (`../../assets/pitch/01-0.png` is the same moment full screen, if you prefer to open maximized and only then split the windows.)
- Keep your Loom camera bubble visible at the bottom. During the introduction, join as **Bruce** with the class code shown on the dashboard and finish on the layout in `../../assets/pitch/01-2.png`.
- Open `assets/grade/5th/math` in Finder, ready to select its three PDF worksheets (Classifying angles, Multiplying by 10/100/1,000, Build a 5-digit number).
- Test mission generation before recording. Keep the live generation in one take and use its waiting time to explain what Mia is building.
- Confirm that the generated preview shows Gemini output, **not** the local fallback banner visible in `../../assets/pitch/02-4.png` ("Fallback could not inspect attachments directly"). The `02-3`/`02-4` screenshots are from a fallback run — retake them if there is time.
- Same fallback run explains the two mission names on screen: the teacher types **Grade 5 Math Skills Quest**, but the generated preview title is **Mathematics Review Mission**, so that is what the student picker shows in `../../assets/pitch/03-2.png`. Regenerate before recording so both names match.
- Use one intentionally wrong answer during the student demo to reveal Mia's adaptive loop.
- Keep the browser zoom high enough for the mission, tutor response, and teacher report to be readable.
- Section 0:35–1:25 is the tightest (live generation + 3 uploads). Rehearse it with a timer.

## 0:00–0:35 — The idea

_[Screen: start with the layout below — student join modal on the left, teacher dashboard on the right, Loom camera bubble at the bottom. While speaking, enter the visible class code, type "Bruce," and click "Enter class."]_

<p align="center"><img src="../../assets/pitch/01-1.png" width="600" alt="Student join modal on the left, teacher dashboard with the class card and 6-digit code on the right"></p>

Hi, my name is Denis. I believe teachers and AI can work together to help students learn better — and have fun doing it.

I've worked in tech for 12 years, and I built ClassGame alone for this hackathon — for desktop and mobile.

On the left, Bruce joins with the six-digit code from the teacher's dashboard on the right. His 3D classroom and the dashboard are now connected.

But how does it actually work?

_[Screen: finish this section with Bruce inside the room, as below.]_

<p align="center"><img src="../../assets/pitch/01-2.png" width="600" alt="Bruce standing in the 3D classroom next to the teacher dashboard"></p>

## 0:35–1:25 — Emily prepares the next lesson

_[Screen: click "Open" on the class card, maximize the teacher window, then point to the class insights.]_

Meet Emily, a fifth-grade math teacher. Past activities show her class needs more help with place value and types of angles.

_[Screen: click "New mission." The empty builder asks for a name, a learning goal, and a teacher prompt.]_

<p align="center"><img src="../../assets/pitch/02-1.png" width="600" alt="Empty Create mission modal with fields for mission name, learning goal, teacher prompt, and a file upload area"></p>

_[Screen: fill the three fields, attach all three PDFs from `assets/grade/5th/math`, then click "Generate preview."]_

<p align="center"><img src="../../assets/pitch/02-2.png" width="600" alt="Create mission modal filled in with the Grade 5 Math Skills Quest name, learning goal, teacher prompt, and the three attached PDFs"></p>

Emily adds three real PDFs from her class. Mia — the Multimodal Intelligence Agent powered by Gemini inside ClassGame — reads the files and writes three questions.

_[Screen: when the preview lands, scroll the right pane to question 1 and read its four choices and expected answer.]_

<p align="center"><img src="../../assets/pitch/02-3.png" width="600" alt="Edit mission panel with the generated preview on the right showing question 1, its four choices, and the expected answer"></p>

_[Screen: open "Flow," zoom in until the labels are readable, and briefly follow the map from top to bottom.]_

<p align="center"><img src="../../assets/pitch/02-4.png" width="600" alt="Flow tab showing the lesson map: teacher starts mission, Mia builds preview, quick diagnostic, guided practice"></p>

In Preview, Emily checks every question and answer. In Flow, she sees the lesson map: a quick check, guided practice, reflection, extra help when needed, and completion. She can edit everything before students see it.

_[Screen: click "Approve and assign to class," then switch to the student tab.]_

When Emily approves, the mission appears for her class in real time.

## 1:25–2:35 — Bruce plays with Mia

_[Screen: return to the split view and keep the student window unchanged. Pause long enough to show the glowing circle appear beside Bruce's desk, as below.]_

<p align="center"><img src="../../assets/pitch/03-1.png" width="600" alt="Glowing circle beside Bruce's desk in the 3D classroom while the dashboard lists the mission as Current and Shared"></p>

Bruce does not refresh. When Emily publishes, a glowing circle appears next to his desk. He walks to it, presses Start, and chooses the new mission.

_[Screen: walk into the circle and open the mission picker, as below. Then maximize the student window and press Start.]_

<p align="center"><img src="../../assets/pitch/03-2.png" width="600" alt="Choose a mission panel in the 3D classroom showing the new mission with its summary, duration, and Start button"></p>

This is more than a quiz. Mia remembers every step.

_[Screen: choose an intentionally wrong answer. Show the "why" prompt, select or type a reason, then let Mia's explanation land — the dashboard on the right streams the same exchange live.]_

<p align="center"><img src="../../assets/pitch/03-3.png" width="600" alt="Student question board on the left; on the right, the teacher's Conversation tab showing Bruce's reason, Mia's step-by-step explanation, and the AI conclusion for the teacher"></p>

When Bruce makes a mistake, Mia does not show the answer. First, she asks why. Bruce can pick a reason or write his own.

Before she replies, Mia loads parts of the PDF, Bruce's answers, past messages, and long-term memory. She explains with simple steps or with pictures, based on what helps him.

_[Screen: show the explanation and tap one feedback option.]_

Bruce says if the help was useful, confusing, too easy, or too hard. If he comes back later, the game opens at the same step.

## 2:35–3:25 — Bruce's game becomes classroom evidence

_[Screen: switch to the class overview, then open a student's details and show the tabs Summary, Learning flow, Conversation, and Memory. Start on "Learning flow" while Bruce is still playing.]_

<p align="center"><img src="../../assets/pitch/03-4.png" width="600" alt="Learning flow tab drawing Bruce's attempt live: mission start, answer, misconception, and feedback nodes with evidence counts"></p>

Back in the dashboard, Bruce's status changes live from playing to completed. Emily can also see who has not started, or who left early.

_[Screen: let Bruce finish the last question, then return to Conversation and scroll to Mia's mission report.]_

<p align="center"><img src="../../assets/pitch/03-5.png" width="600" alt="Student panel marked Completed with Insights ready, showing Bruce's correct answer and Mia's finished learning report for the teacher"></p>

Mia uses Bruce's answers, reasons, and feedback to write a learning report. Emily sees what he does well, what needs more work, his path, and which parts of the PDF were used.

_[Screen: go back to the class Overview tab so the whole class is visible, not just Bruce.]_

<p align="center"><img src="../../assets/pitch/04-1.png" width="600" alt="Class overview banner saying one student may need a check-in, with the class progress ring and a chart of where every student is in today's mission"></p>

This memory stays for future missions. When several students have the same problem, ClassGame helps Emily see what the class needs and plan the next lesson. This closes the loop.

## 3:25–3:48 — Built as a real agent

_[Screen: open the Architecture page in the sidebar (`/dashboard/architecture`) and let the diagram fill the window while you speak.]_

<p align="center"><img src="../../assets/pitch/05-1.png" width="600" alt="ClassGame production architecture diagram: browser clients, Cloud Run API, the Mia tutor, learning, and mission agents, Gemini, Firebase, Cloud Tasks, Redis, and PostHog"></p>

ClassGame uses Gemini on Vertex AI, with a Go API on Cloud Run. Firestore keeps live data and long-term memory. Cloud Tasks runs background jobs, and Redis limits AI requests. Private teacher notes never reach students. If AI is not available, backup logic keeps the game working.

## 3:48–4:00 — Close

_[Screen: return to the 3D classroom; camera remains visible.]_

ClassGame is optional homework that takes notes for the teacher. Mia learns with every student and brings that learning back to class.

Thank you.

---

## Screenshot index

| File       | Section | What it shows                                                  |
| ---------- | ------- | -------------------------------------------------------------- |
| `01-0.png` | 0:00    | Join modal full screen, code and name filled in                |
| `01-1.png` | 0:00    | Split view: join modal + teacher dashboard with the class code |
| `01-2.png` | 0:00    | Bruce inside the 3D classroom, class now "Needs publishing"    |
| `02-1.png` | 0:35    | Empty mission builder                                          |
| `02-2.png` | 0:35    | Builder filled in with the three PDFs attached                 |
| `02-3.png` | 0:35    | Generated preview: question 1, choices, expected answer        |
| `02-4.png` | 0:35    | Flow tab with the lesson map (fallback banner — retake)        |
| `03-1.png` | 1:25    | Glowing circle appears; mission listed as Current / Shared     |
| `03-2.png` | 1:25    | Student mission picker with Start                              |
| `03-3.png` | 1:25    | Wrong answer, Bruce's reason, Mia's step-by-step reply         |
| `03-4.png` | 2:35    | Learning flow: start → answer → misconception → feedback       |
| `03-5.png` | 2:35    | Completed, insights ready, Mia's mission report                |
| `04-1.png` | 2:35    | Class overview: who needs a check-in, who is where             |
| `05-1.png` | 3:25    | In-app production architecture diagram                         |

## Delivery notes

- Pause at every period. Short sentences are on purpose — do not join them.
- Words avoided on purpose (hard for PT-BR speakers): _worksheet_ → **PDF**, _strengths_ → **what he does well**, _through_ → **on / with**, _artificial intelligence_ → **AI**, _one thousand_ (cut from the topic list).
- Stress the key words: **teachers and AI**, **three real PDFs**, **asks why**, **remembers**, **closes the loop**, **takes notes for the teacher**.
- Word budget per section: 0:00 ≈ 70 · 0:35 ≈ 90 · 1:25 ≈ 110 · 2:35 ≈ 85 · 3:25 ≈ 50 · Close ≈ 25.

Emily uploads a PDF. Mia, our AI agent, calls Gemini. Gemini reads the PDF and suggests three questions

In Preview, Emily checks every question and answer. And in Flow, she sees how our AI will evaluate her students' answers. She can edit everything before approve

and now after Emily approves the mission, a glowing circle appears in Bruce's game. He walks to it, presses Start, and selects the new mission

The student gets the first question right. We send the answer to Gemini while the student moves to the next question.
