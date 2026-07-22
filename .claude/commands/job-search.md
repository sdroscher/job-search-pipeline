# job-search

Commands for managing your job search pipeline. All commands call `http://localhost:8080` (override with `JOB_PIPELINE_URL` env var).

## Variables

Read at skill load time:
- `BASE_URL`: `${JOB_PIPELINE_URL:-http://localhost:8080}`
- `OUTPUT_DIR`: `${OUTPUT_DIR:-./output}`

## Tool usage

- **GET requests** → use the `WebFetch` tool (no confirmation prompt)
- **POST / PUT / PATCH / DELETE** → use `curl` via Bash (pre-approved for localhost)
- **Writing files** → use the `Write` tool
- **Never** use Python, Node, or any other interpreter to call the API

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
4. PUT to `$BASE_URL/api/profile` with `Content-Type: application/json`:
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
     "writing_voice_md": "<notes or null>"
   }
   ```
   All fields except `resume_md` are optional (omit or null if not provided).
5. Confirm: "Profile saved. Open http://localhost:8080 to see your board."

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
5. Generate a slug ID: lowercase company + role keywords, hyphens, no spaces (e.g. `temporal-senior-swe-observability`)
6. POST to `$BASE_URL/api/jobs` with all fields.
7. POST to `$BASE_URL/api/jobs/<id>/activity`: `{"action": "Evaluated", "notes": "Added via /job-search add"}`
8. Confirm: "Added **<Company>** — <Role> (fitScore: <N>/10 <emoji>)"

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

1. GET `$BASE_URL/api/jobs/<job-id>`
2. GET `$BASE_URL/api/profile` (use writing_voice_md if present, cover_letter_sample if present)
3. Generate a 4-paragraph cover letter:
   - P1: What drew the user to THIS company specifically — named values, specific tech, remote-first, etc. 2–3 sentences. Direct, warm. No clichés.
   - P2: "My background maps directly to [team/role]" — one specific achievement story grounding the connection.
   - P3: Three numbered achievements from the user's profile most relevant to the JD. Specific and measurable where possible.
   - P4: Availability and closing invitation.
   - Voice: modern professional, not corporate or hypey. Confident through specifics, not adjectives. Use the writing_voice_md guide if provided.
4. Write to `$OUTPUT_DIR/cover-letter-<company>-<role>.md`
5. POST artifact + activity (same pattern as resume command)
6. Confirm: "Cover letter written to <filepath>"

---

## /job-search prep <job-id>

Generate interview preparation notes.

1. GET `$BASE_URL/api/jobs/<job-id>`
2. GET `$BASE_URL/api/profile`
3. Generate a prep document:
   - **Likely behavioral questions** (5–7) with a suggested STAR story from the user's experience for each
   - **Likely technical questions** based on the role's stack and responsibilities (5–7)
   - **Research checklist**: company news, recent blog posts, specific team/product context
   - **Questions to ask them** (5–7) — about team structure, on-call, tech debt, growth
4. Write to `$OUTPUT_DIR/prep-<company>-<role>.md`
5. POST activity: `{"action": "Interview prep generated"}`
6. Confirm: "Prep notes written to <filepath>"

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
