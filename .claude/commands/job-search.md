# job-search

Commands for managing your job search pipeline. All commands call `http://localhost:8080` (override with `JOB_PIPELINE_URL` env var).

## Variables

Read at skill load time:
- `BASE_URL`: `${JOB_PIPELINE_URL:-http://localhost:8080}`
- `OUTPUT_DIR`: `${OUTPUT_DIR:-./output}`

## Tool usage

- **GET requests** → use the `WebFetch` tool (no confirmation prompt)
- **POST / PUT / PATCH / DELETE** → use `curl` via Bash exactly as shown below
- **Writing files** → use the `Write` tool
- **Never** use Python, Node, or any other interpreter to call the API

### curl rules (important — deviating causes confirmation prompts)

Always use this exact form — no `cd` prefix, no temp files, no pipes to `jq`:

```bash
curl -s -X POST "$BASE_URL/api/jobs" \
  -H "Content-Type: application/json" \
  -d '{"id":"...","company":"..."}'
```

Read the JSON response directly from curl's stdout. Do not redirect to files or pipe through `jq`.

---

## /job-search init

Set up your profile. Run this once.

1. Ask the user to paste their resume (markdown or plain text), or provide a file path. If a path, read it with the Read tool.
2. Ask: "Do you have a sample cover letter to upload? (optional)"
3. Ask the following questions in sequence — don't batch them:
   - Desired salary range (number, in CAD unless otherwise specified)? Enter min and target.
   - Remote preference? (remote-only / hybrid-ok / open)
   - Location (city, country)?
   - Industries or company types of interest?
   - Preferred tech stack?
   - Green flags — what excites you in a role?
   - Red flags — deal-breakers?
   - Writing voice notes — anything specific about your cover letter style? (optional)
4. Ask about achievements for cover letters. Say:

   > "Last one: do you have a bank of pre-written achievement bullets for cover letters?
   > These are the specific, quantified accomplishments you'd pull from for Para 2/3 —
   > e.g. 'Architected a Notifications platform delivering 1,700+ notifications/sec to 90M+ users.'
   > Organising them by theme (Scale, Architecture, Team, etc.) helps the cover-letter command
   > pick the most JD-relevant ones.
   >
   > You can paste them now (markdown is fine), provide a file path, or say 'skip'."

   If they provide a path, read it with the Read tool. If they skip, set `achievements_md` to null.

5. PUT to `$BASE_URL/api/profile` with `Content-Type: application/json`:
   ```json
   {
     "resume_md": "<resume text>",
     "cover_letter_sample": "<sample or null>",
     "salary_min": <number or null>,
     "salary_max": <number or null>,
     "salary_target": <number or null>,
     "remote_pref": "<remote-only|hybrid-ok|open>",
     "location": "<city, country>",
     "industries": "<comma-separated or freeform>",
     "tech_prefs": "<comma-separated or freeform>",
     "green_flags": "<freeform>",
     "red_flags": "<freeform>",
     "writing_voice_md": "<notes or null>",
     "achievements_md": "<achievement bank markdown or null>"
   }
   ```
   All fields except `resume_md` are optional (omit or null if not provided).
6. Confirm: "Profile saved. Open http://localhost:8080 to see your board."

---

## /job-search add <url-or-paste>

Parse a job posting, evaluate fit, add to board.

1. If the argument looks like a URL, POST `{"url": "<url>"}` to `$BASE_URL/api/parse`.
   - If the parse returns an error or BodyMD is under 200 characters, tell the user: "Couldn't extract enough content from that URL. Paste the job description below." Then use their pasted text as the raw JD.
2. GET `$BASE_URL/api/profile` to load the user profile.
3. Fetch the company's careers or about page to find named values, mission, and tech stack. Use WebFetch.
4. Evaluate fit against the profile:
   - fitScore 1–10: weight salary alignment (25%), remote/location match (20%), tech stack match (20%), green/red flag match (20%), role seniority match (15%)
   - verdict: green (8–10), yellow (5–7), red (1–4)
   - 3–5 positives (specific, not generic)
   - 3–5 concerns (specific, flag on-call, low salary, travel, non-Go stack, etc.)
   - one-paragraph summary (2–3 sentences, what's notable about this role for THIS user)
   - company_values: [{name, description}] from careers page
4.5. **Agency detection — find the real listing:**
   - Check `result.source` from the parse response. Known agency/aggregator values: `"Jobgether"`, `"Jobright"`, `"LinkedIn"`, `"Indeed"`, `"Glassdoor"`.
   - If `result.source` is one of these:
     a. Note the company name (`result.company`) and role (`result.title`) from the parsed data.
     b. Use WebSearch to find the company's direct job listing: search for `"{company}" "{role}" careers apply site:greenhouse.io OR site:ashbyhq.com OR site:lever.co OR site:bamboohr.com OR site:smartrecruiters.com` (or search directly on the company's careers site if known).
     c. If a direct ATS link is found, tell the user: "**This listing is from {source} (an aggregator).** I found what looks like the direct posting at `{url}` — want me to use that instead? (Say 'yes', 'no', or paste a different URL.)"
     d. If the user says yes: re-run the parse with the direct URL (POST `$BASE_URL/api/parse` with the new URL) and continue from step 2 with the new result.
     e. If the user says no or no direct link is found: continue with the aggregator's data. Add a note to `my_notes`: "Source: {source} aggregator — direct listing not confirmed."
   - If `result.source` is NOT an agency value, skip this step entirely.
5. **Before finalizing — gather missing info and context (one turn, ask everything at once):**
   - **Missing info:** If salary or location is absent from the parsed data, flag it: "I couldn't find [salary / location] in the posting — do you have that? (Say 'skip' to continue without it.)" If they provide it, fold it into the evaluation before scoring.
   - **Additional context:** "Any context I should know? e.g., recruiter reach-out, internal referral, found it yourself, heard from a friend — or skip." Store the answer in `my_notes`.
   - **Networking:** "Do you have a contact at <Company>?" If yes, ask for their name and role, store in the `networking` field (e.g., `"Jane Doe – Senior Engineer"`), and add +0.5 to the raw fit score (cap at 10).
   Ask all three in one message. Do not make the user wait through three separate turns.
6. Generate a short, memorable slug ID: company name + the 2–3 most distinctive words from the role title, all lowercase, hyphen-separated, 3–5 words total. The user will type this ID in future commands, so make it easy to remember and short to type. Examples: `stripe-staff-swe`, `grafana-senior-backend`, `planhub-senior-php`, `temporal-swe-observability`. Do NOT append the full role title verbatim — distil it.
7. POST to `$BASE_URL/api/jobs` with all fields.
8. POST to `$BASE_URL/api/jobs/<id>/activity`: `{"action": "Evaluated", "notes": "Added via /job-search add"}`
9. Confirm with the job ID clearly visible:
   > Added **<Company>** — <Role> · ID: `<slug-id>` · fitScore: <N>/10 <emoji>
   > Use `<slug-id>` in future commands, e.g. `/job-search resume <slug-id>`
10. Offer follow-ups in one message:
    - "Want a tailored resume or cover letter? Say so and I'll run the full command." If the user says yes, execute the `/job-search resume` or `/job-search cover-letter` steps from this file verbatim — including the pre-flight questions for cover letters. Never generate artifacts inline without following those steps.
    - If a networking contact was recorded: "Want me to draft a reach-out to <contact name>? Run `/job-search reach-out <slug-id>` or say 'yes' now."

---

## /job-search reach-out <job-id>

Draft a personalized reach-out message to a networking contact at the company.

1. GET `$BASE_URL/api/jobs/<job-id>`. If `networking` is already populated, use that as the contact info.
2. GET `$BASE_URL/api/profile`.
3. Ask in one turn (anything not already known from the `networking` field):
   - "Who are you reaching out to, and how do you know them? (e.g., ex-colleague, met at a conference, mutual connection, cold reach-out)"
   - "What do you want from this? (e.g., intro call, referral for this specific role, just keeping the relationship warm)"
   - "Email or LinkedIn message?"
4. Draft the message:
   - LinkedIn: under 150 words. Email: under 250.
   - Open with the specific connection or shared context — not "I hope this finds you well."
   - One sentence on what Simon is exploring and why this company interests him specifically (draw from `summary` and `company_values`). Do NOT say "I saw a job posting."
   - Ask for exactly one low-friction thing: a 20-minute call, their honest read on the team, a referral if the relationship warrants it.
   - Warm and direct. No corporate filler. No em-dashes. No AI-marker phrases.
5. POST activity: `{"action": "Reach-out drafted", "notes": "<contact name> via <email|LinkedIn>"}`
6. Show the draft and offer one round of revision.

---

## /job-search resume <job-id>

Generate a tailored resume. Write to filesystem.

1. GET `$BASE_URL/api/jobs/<job-id>`
2. GET `$BASE_URL/api/profile`
3. Generate a tailored resume in markdown:
   - Reorder experience bullets so the most JD-relevant ones come first in each role
   - Adjust the summary/objective to use language from the JD
   - Surface specific technologies mentioned in the JD requirements
   - Do NOT invent experience — only reorder and reframe
   - No em-dashes (`—`), en-dashes used as dashes, or AI-marker phrases ("leverage", "utilize", "dynamic", "proven track record", "passionate")
   - Bullet points: action verb, specific outcome, no filler words
4. Write the markdown to `$OUTPUT_DIR/resume-<company>-<role>.md` using the Write tool
5. POST to `$BASE_URL/api/jobs/<job-id>/artifacts`:
   ```json
   {
     "type": "resume",
     "filepath": "<full path>",
     "profile_hash": "<profile.profile_hash>"
   }
   ```
6. POST activity: `{"action": "Resume generated", "notes": "<filepath>"}`
7. Confirm: "Resume written to <filepath>"

---

## /job-search cover-letter <job-id>

Generate a cover letter in the user's voice. Write to filesystem.

1. GET `$BASE_URL/api/jobs/<job-id>` and GET `$BASE_URL/api/profile` in parallel.
2. Before writing anything, ask the user two questions (ask both at once, don't split into separate turns):
   - "What specifically drew you to this role or company?" — get something personal and concrete, not just a restatement of the job description. If they say "I don't know" or give a generic answer, push back once with a specific prompt: "Is there something about their product, stack, scale, team structure, or mission that caught your attention?"
   - "Is there anything you want to make sure the cover letter highlights or avoids?"
3. Generate a 4-paragraph cover letter using the user's answer to ground P1:
   - **P1 (Hook):** Open with what the user told you drew them to this role. Make it feel personal and specific — not "I was excited to see this posting." Write from their perspective, in their voice. 2–3 sentences.
   - **P2 (Connection):** One concrete achievement story that connects their background to the team's actual work. This should not duplicate a bullet point from the resume — go deeper, give context, explain the impact in a sentence the resume doesn't have room for.
   - **P3 (Evidence):** Two or three specific accomplishments most relevant to the JD. **If `profile.achievements_md` is populated, select bullets from it** — pick the 2–3 that best match what the JD is asking for (tech stack, scale, leadership, etc.). Adapt lightly for sentence flow but do not change the facts or invent new details. If `achievements_md` is empty, fall back to the resume, but find accomplishments that go beyond what the resume bullet says — quantify further, name the decision, describe the outcome.
   - **P4 (Close):** Availability and a direct, warm invitation to talk. One or two sentences. No "I look forward to hearing from you at your earliest convenience."
4. Style rules — enforce these without exception:
   - No em-dashes (`—`). Use a comma, period, or rewrite the sentence.
   - No en-dashes used as em-dashes. No ellipses for effect.
   - No AI-marker phrases: "I am excited to", "I am passionate about", "I would be remiss", "delve", "leverage" (as a verb), "utilize", "I am writing to express", "dynamic", "synergy", "proven track record".
   - No clichés: "hit the ground running", "wear many hats", "go-getter", "team player".
   - Active voice. Sentences under 25 words where possible. Confident without being breathless.
   - Use the `writing_voice_md` guide if provided. Use the `cover_letter_sample` only to calibrate tone and sentence rhythm — do not copy phrases.
5. Check for duplication before writing:
   - The cover letter must not open with the same sentence structure as the resume summary.
   - P3 achievements should add context beyond what the resume bullet already says — if a bullet says "reduced latency by 40%", the cover letter can say what decision led to that, not just repeat the number.
6. Write to `$OUTPUT_DIR/cover-letter-<company>-<role>.md`
7. POST artifact + activity (same pattern as resume command)
8. Confirm: "Cover letter written to <filepath>"

---

## /job-search prep <job-id>

Generate interview preparation notes.

1. GET `$BASE_URL/api/jobs/<job-id>`
2. GET `$BASE_URL/api/profile`
3. **Before generating — ask for context in one turn:**
   - "Do you know who you'll be interviewing with? If so, share their names and roles."
   - "Can you paste anything about them — a LinkedIn bio, a post or article they wrote, their GitHub? Even just their job title helps tailor the behavioral questions."
   - "What do you know about the format? (e.g., number of rounds, 1:1 vs panel, technical screen, system design, take-home)"
   Wait for the user to answer. If they say "skip" or have nothing, proceed with what you have from the job record.
4. Generate a prep document, using any interviewer context to personalise:
   - **Likely behavioral questions** (5–7) — if interviewers were named and context provided, tailor at least 2 questions to what you know about them (e.g., a question on their known technical focus area or a topic from a post they wrote)
   - **Suggested STAR story** for each behavioral question, drawn from the user's profile and experience
   - **Likely technical questions** based on the role's stack and responsibilities (5–7)
   - **Research checklist**: company news, recent blog posts, specific team/product context
   - **Questions to ask them** (5–7) — at least 2 must be grounded in the company's specific values or culture from the `company_values` field (not generic); the rest cover team structure, on-call, tech debt, growth trajectory
5. Write to `$OUTPUT_DIR/prep-<company>-<role>.md`
6. POST activity: `{"action": "Interview prep generated"}`
7. Confirm: "Prep notes written to <filepath>"

---

## /job-search eval <job-id>

Re-evaluate fit against the current profile. Use when: salary was revealed, profile was updated, or new information emerged.

1. GET `$BASE_URL/api/jobs/<job-id>`
2. GET `$BASE_URL/api/profile`
3. Re-run fit evaluation (same scoring rubric as `add`)
4. PATCH `$BASE_URL/api/jobs/<job-id>`:
   ```json
   {"fit_score": <n>, "verdict": "<v>", "positives": "<json>", "concerns": "<json>", "summary": "<s>"}
   ```
5. POST activity: `{"action": "Re-evaluated", "notes": "fitScore <old> → <new>"}`
6. Confirm: "Re-evaluated: fitScore now <N>/10 <emoji>"

---

## /job-search compare <id1> <id2>

Side-by-side comparison of two jobs to help decide which to prioritize.

1. GET `$BASE_URL/api/jobs/<id1>` and GET `$BASE_URL/api/jobs/<id2>` in parallel.
2. GET `$BASE_URL/api/profile`.
3. Build a side-by-side comparison table:

   | Dimension | <Company A> | <Company B> |
   |-----------|-------------|-------------|
   | Role | | |
   | Fit Score | | |
   | Verdict | | |
   | Salary | | |
   | Remote | | |
   | Stage | | |

4. Highlight the key positives and concerns for each role (bullet list per job, 3 each max).
5. Write a **Recommendation** paragraph (3–5 sentences) naming which role to prioritize next and why, based on fit score, salary alignment to profile target, remote match, and which is likely to move faster based on stage.
6. Output everything as formatted markdown in the conversation — do not write to a file or call any API.
