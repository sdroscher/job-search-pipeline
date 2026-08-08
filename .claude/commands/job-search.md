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

# API contract

**Read this section before your first write call. It is the only authoritative source for field names — do not infer a field name from a response you saw earlier, from a database column, or from what the field is called in this document's prose.**

## Three rules

1. **Never invent or guess a field name.** Only send names that appear in the tables below. If the field you want isn't listed, the endpoint does not accept it — say so rather than sending a plausible-looking alternative.
2. **A 400 `json: unknown field "x"` means you used the wrong name.** The server rejects unknown fields instead of dropping them. Come back to this section, find the correct name, and resend. Do not strip the field and retry without it — that discards data the user gave you.
3. **Verify every write.** `POST /api/jobs` and `PATCH /api/jobs/<id>` both return the full saved job. Read it and confirm the fields you set came back with the values you sent. If `role`, `company`, or `fit_score` came back empty or wrong, fix it with a PATCH before telling the user the job was added.

## Reading a posting vs. storing it — the names differ

`POST /api/parse` returns a *parsed posting*. `POST /api/jobs` stores a *job record*. They are different schemas. Translate every field:

| `/api/parse` returns | → `/api/jobs` expects | Note |
|---|---|---|
| `title` | `role` | **Most common mistake.** There is no `title` field on a job. |
| `company` | `company` | Same name. |
| `salary_raw` | `salary` | Free text, e.g. `"$180k–$220k CAD"`. |
| `body_md` | `raw_jd` | The full job description markdown. |
| `source` | `source` | Same name. Also used for agency detection in `add` step 4.5. |
| `source_url` | `source_url` | Same name. |
| `location` | — | No `location` field on a job. Fold it into `role_details` or `my_notes`. |
| `department` | — | No matching field. Fold into `role_details` if useful. |
| `benefits` | — | An array. No matching field. Fold into `role_details` if useful. |

There is no `salary_min` in the parse response — derive it from `salary_raw` yourself and send it as an integer.

## POST /api/jobs — create a job

`id`, `company`, and `role` are required; sending any of them empty returns 400. Everything else is optional.

<!-- schema:createJobRequest -->

| field | type | required | notes |
|---|---|---|---|
| `id` | string | **yes** | The slug from `add` step 6. |
| `company` | string | **yes** | |
| `role` | string | **yes** | The job title. Not `title`. |
| `stage` | string | no | Defaults to `Evaluated`. Must be one of the stage values below — an unrecognised stage puts the job in no board column. |
| `verdict` | string | no | `green` / `yellow` / `red`. Defaults to `yellow`. |
| `salary` | string | no | Free text as written in the posting. |
| `salary_min` | integer | no | Not a string. |
| `remote` | string | no | |
| `source` | string | no | |
| `source_url` | string | no | |
| `raw_jd` | string | no | Full JD markdown. |
| `added` | string | no | `YYYY-MM-DD`. Defaults to today. |
| `last_activity` | string | no | `YYYY-MM-DD`. Defaults to today. |
| `fit_score` | integer | no | 1–10. Not a string, not a float. |
| `summary` | string | no | |
| `positives` | string | no | A JSON **string**, not an array: `"[\"a\",\"b\"]"`. |
| `concerns` | string | no | Same: JSON-encoded string, not an array. |
| `my_notes` | string | no | |
| `company_values` | string | no | Same: JSON-encoded string of `[{name, description}]`. |
| `networking` | string | no | e.g. `"Jane Doe – Senior Engineer"`. |
| `role_details` | string | no | |

### Stage values

The board renders one column per active stage and groups the rest under closed. A `stage` outside these lists is stored but appears nowhere in the UI, so never invent one.

**Active:** `Evaluated`, `Applied`, `AI Assessment`, `Screening`, `Interviewing`, `Final Round`, `Offer`
**Closed:** `Rejected`, `Listing Withdrawn`, `Declined`, `Won't Apply`

## PATCH /api/jobs/&lt;id&gt; — update a job

Send only the fields you are changing; omitted fields keep their current values. The job id goes in the URL. Sending a different `id` in the body is a 400 — job ids are immutable.

One field is not preserved: **every PATCH sets `last_activity` to today**, including a pure correction like fixing a typo in `company`. The board shows that date, so a two-week-old application will look like it was touched today. Mention it if the user might care.

<!-- schema:UpdateJobParams -->

| field | type | notes |
|---|---|---|
| `company` | string | |
| `role` | string | |
| `stage` | string | |
| `verdict` | string | |
| `salary` | string | |
| `salary_min` | integer | |
| `remote` | string | |
| `source` | string | |
| `source_url` | string | Use this if the direct ATS listing turns up after the job was added from an aggregator. |
| `raw_jd` | string | |
| `fit_score` | integer | |
| `summary` | string | |
| `positives` | string | JSON-encoded string. |
| `concerns` | string | JSON-encoded string. |
| `my_notes` | string | |
| `company_values` | string | JSON-encoded string. |
| `networking` | string | |
| `role_details` | string | |

Every field above is fixable in place, so PATCH rather than recreating. **`id` cannot be changed, and `DELETE /api/jobs/<id>` is a soft delete** — it moves the job to the "Won't Apply" stage and keeps the row, so the id stays taken and a recreate returns `409`. If the id itself is wrong, tell the user; don't try to work around it.

## PUT /api/profile — save the profile

This is a full replace, not a merge: every field you omit is set to null. To change one field, GET the profile first, then resend every field you want to keep along with the changed one.

`resume_md` is required on every write, including a one-field update — a body without it is a 400 rather than a blanked resume.

A GET response can be sent straight back: `id`, `profile_hash`, and `updated_at` are accepted and ignored, so you don't have to strip them. They are server-owned — `profile_hash` is recomputed from `resume_md` on every write. Every *other* unknown field is still a 400, so a wrong name like `achievement_bank` will be caught.

<!-- schema:profileRequest -->

| field | type | required | notes |
|---|---|---|---|
| `resume_md` | string | **yes** | |
| `cover_letter_sample` | string | no | |
| `salary_min` | integer | no | |
| `salary_max` | integer | no | |
| `salary_target` | integer | no | |
| `remote_pref` | string | no | `remote-only` / `hybrid-ok` / `open`. |
| `location` | string | no | |
| `industries` | string | no | |
| `green_flags` | string | no | |
| `red_flags` | string | no | |
| `tech_prefs` | string | no | |
| `writing_voice_md` | string | no | |
| `achievements_md` | string | no | The achievement bank from `init` step 4. |
| `career_notes_md` | string | no | The career story blocks from `init` step 1.5. |

## POST /api/jobs/&lt;id&gt;/activity

<!-- schema:createActivityRequest -->

| field | type | required | notes |
|---|---|---|---|
| `action` | string | **yes** | |
| `notes` | string | no | Plural. Not `note`. |
| `date` | string | no | `YYYY-MM-DD`. Defaults to today. |

## POST /api/jobs/&lt;id&gt;/artifacts

<!-- schema:createArtifactRequest -->

| field | type | required | notes |
|---|---|---|---|
| `type` | string | **yes** | `resume` / `cover-letter` / `prep`. |
| `filepath` | string | **yes** | Absolute path. Not `path`. Must be inside `$OUTPUT_DIR`. |
| `profile_hash` | string | **yes** | Copy `profile_hash` from `GET /api/profile`. |

## POST /api/parse

<!-- schema:parseRequest -->

| field | type | required | notes |
|---|---|---|---|
| `url` | string | **yes** | Not `link`, not `source_url`. |

Response fields are listed in the translation table above.

> Maintaining this section: the field lists are checked against the Go request
> structs by `TestCommandDocMatchesSchema` in `internal/api/contract_test.go`.
> If you change an API struct, that test tells you which table to update.

---

## /job-search init

Set up your profile, or fill in the gaps in an existing one. Safe to re-run.

0. **Load the existing profile first.** GET `$BASE_URL/api/profile`.

   - **404** — first run. Go to step 1 and ask everything.
   - **200** — a profile exists. Build a list of which fields are already populated and which are null or empty. Then show the user where they stand and ask how to proceed, in one message:

     > "You already have a profile. Here's what's set and what's missing:
     >
     > **Set:** resume, salary range, remote preference, location, ...
     > **Missing:** achievement bank, career notes, writing voice
     >
     > Want me to (a) just fill in what's missing, (b) go through everything again, or (c) update
     > specific fields — tell me which?"

     Use the real field lists, not this example. Name the missing fields in plain language: `achievements_md` is "the achievement bank", `career_notes_md` is "the career story notes", `cover_letter_sample` is "a sample cover letter".

   - **(a) fill gaps** — run only the steps below whose fields are currently null or empty. Skip the rest entirely; do not re-ask a question the profile already answers.
   - **(b) redo everything** — run every step, but show the current value with each question so the user can say "keep" instead of retyping it.
   - **(c) specific fields** — run only the steps for the fields they named.

   In all three cases, carry the untouched values through to the PUT in step 5 unchanged. Skipping a question means keeping its current value, never nulling it.

   Note for existing profiles: `achievements_md` and `career_notes_md` were silently dropped by the API before 2026-08-07. If a profile predates that and both are null, the user very likely did answer those questions and the answers were lost. Say so rather than implying they skipped them.

1. Ask the user to paste their resume (markdown or plain text), or provide a file path. If a path, read it with the Read tool. On a re-run where `resume_md` is already set and the user chose (a), skip this and keep the stored resume; steps 1.5 and 3 below still read it.

1.5. **Career context — build the depth that makes tailoring work.** Skip this step entirely if `career_notes_md` is already populated and the user chose (a). If you are filling a gap rather than running a first-time setup, drop "Before I ask the remaining profile questions" from the opening and say you're filling in the one missing piece. Once you have the resume, say:

   > "Got it. Before I ask the remaining profile questions, I'd like to get the story behind a few
   > of your strongest pieces of work. This is what the cover-letter command draws on for the
   > narrative paragraph, and what lets the resume command pick the right *angle* on each bullet
   > rather than just matching keywords.
   >
   > I'll pick 2–3 things from your resume and ask about each one. Takes about 5–10 minutes.
   > Want to do this now, or skip and come back later?"

   If they say skip (or later), leave `career_notes_md` at its current value — null on a first run, unchanged on a re-run — and continue to step 2.

   If they say yes:

   a. **Select what to dig into.** Read their resume and identify the 2–3 most substantial
      pieces of work — ideally from the most recent role, but pick whatever looks most impactful
      or likely to be relevant to senior roles. Look for: complex systems, large scale, team
      leadership, cross-org influence. Do not pick routine or incremental work.

   b. **For each selected project/achievement, run this sequence (one project at a time):**

      i. Open with: "Tell me about [X]. What was the situation when you picked this up, what
         problem were you actually solving, and what did you end up building or deciding?"
         Wait for their full answer.

      ii. Ask **one targeted follow-up** based on what's still missing:
          - If no scale/numbers: "Can you put numbers on it? Users affected, throughput,
            team size, time saved, before vs. after — any metric that captures the scope."
          - If the decision/tradeoff isn't clear: "What was the key decision you made, and
            what were you choosing between?"
          - If impact is vague: "What changed as a result — for the product, the team, or
            the users?"

      iii. Synthesise what they told you into a structured story block:
           ```
           ## [Role] — [Company] ([dates if known])
           ### [Project name]
           **Context:** [1–2 sentences on the situation]
           **The problem:** [what was broken, missing, or needed]
           **What I did:** [the approach and key decisions]
           **Outcome:** [quantified result + broader impact]
           **Cover letter angle:** [1 sentence — the most compelling way to open a P2 story
             about this for a reader who cares about [scale / architecture / leadership /
             quality — pick whichever fits]]
           ```
           Show it to them and ask: "Does this capture it accurately? Anything to add or change?"
           Apply any edits.

      iv. Move to the next project.

   c. After covering all selected projects, compile the story blocks into `career_notes_md`.

2. Ask: "Do you have a sample cover letter to upload? (optional)" Skip if `cover_letter_sample` is already set and the user chose (a).
3. Ask the following questions in sequence — don't batch them. Skip any whose field is already populated unless the user asked to redo it; when redoing, show the current value and accept "keep".
   - Desired salary range (number, in CAD unless otherwise specified)? Enter min and target.
   - Remote preference? (remote-only / hybrid-ok / open)
   - Location (city, country)?
   - Industries or company types of interest?
   - Preferred tech stack?
   - Green flags — what excites you in a role?
   - Red flags — deal-breakers?
   - Writing voice notes — anything specific about your cover letter style? (optional)
4. Build the achievement bank conversationally. Skip this step entirely if `achievements_md` is already populated and the user chose (a). If you are filling a gap, drop "Last thing" from the opening. Say:

   > "Last thing: let's build a bank of achievement bullets the cover-letter command can pull from.
   > I'll ask about a few areas of your work — just describe what you did in rough terms and I'll
   > help shape each one into a tight, specific bullet. Say 'skip' to skip a category, or 'done'
   > at any point if you've covered enough."

   Then go through each category below **one at a time**, waiting for a response before moving to
   the next. Do not list all categories upfront.

   **Categories to cover (in order):**
   a. **Scale & reliability** — "Tell me about something you built or ran at significant scale —
      high traffic, large user base, complex data volume. What was it, and what were the numbers?"
   b. **Architecture or technical design** — "Was there a system you designed or meaningfully
      redesigned? What was the problem, what did you build, and what changed?"
   c. **Code quality or team uplift** — "Did you improve test coverage, coding standards, processes,
      or the way your team works? What was the before and after?"
   d. **Cross-team or org-level impact** — "Any work that shaped things beyond your immediate team —
      an engineering-wide standard, an initiative you drove, influence on a product direction?"
   e. **Earlier career (optional)** — "Anything from earlier roles worth including for senior roles
      that aren't covered above? Say 'skip' if not."

   **For each response that isn't 'skip':**
   - If numbers or scope are vague, ask one follow-up: "Can you put a number on that? (users
     affected, % improvement, team size, time saved, throughput, etc.)"
   - Once you have enough detail, draft a tight 1–2 sentence bullet in this form:
     action verb + what you built/did + quantified result or impact.
   - Show it and ask: "How does this read? Any changes?" Apply any edits before moving on.

   After all categories, compile the approved bullets into a markdown achievement bank organised
   by category heading. Set this as `achievements_md`.

   If the user says 'skip' to all categories or 'skip' at the start, leave `achievements_md` as it was — null on a first run, unchanged on a re-run. Never overwrite an existing bank with null.

5. PUT to `$BASE_URL/api/profile` with `Content-Type: application/json`. Field names are fixed — see **PUT /api/profile** in the API contract.

   **The PUT replaces the whole profile, so build the body by merging, not by sending only what you collected this run.** Start from the profile you loaded in step 0, overwrite the fields the user just answered, and send the whole thing back. Anything the user skipped keeps the value it already had. You can leave `id`, `profile_hash`, and `updated_at` in the body; the server ignores them.

   Send `achievements_md` and `career_notes_md` here; a profile saved without them leaves `resume` and `cover-letter` with nothing to draw on.
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
     "achievements_md": "<achievement bank markdown or null>",
     "career_notes_md": "<career story blocks markdown or null>"
   }
   ```
   All fields except `resume_md` are optional (omit or null if not provided).
6. Read the PUT response and confirm the fields you just collected came back non-null. If one came back null, the field name was wrong — check the API contract and resend rather than reporting success.
7. Confirm: "Profile saved. Open http://localhost:8080 to see your board." On a gap-fill run, name what you added instead: "Profile updated — added your achievement bank and career notes."

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
   - Use the **POST /api/jobs** table in the API contract for field names. The parsed posting's `title` goes in `role`, `salary_raw` goes in `salary`, `body_md` goes in `raw_jd` — translate every field before sending.
   - `positives`, `concerns`, and `company_values` are JSON-encoded **strings**, not arrays.
7.5. **Verify the write before moving on.** The POST returns the saved job. Check that `role`, `company`, `fit_score`, and `verdict` came back with the values you sent. If any is empty or wrong, PATCH `$BASE_URL/api/jobs/<id>` to correct it. Do not report the job as added until the read-back matches.
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
   - If `profile.career_notes_md` is populated, read it before reordering anything. Use the
     context, decisions, and outcomes it captures to understand *which aspect* of each bullet
     best matches this JD — not just whether the bullet matches keywords. For example: if the
     JD emphasises team leadership and career_notes describes how a project involved turning
     around an underperforming team, lead with that angle even if the resume bullet currently
     leads with a technical metric.
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
   - **P2 (Connection):** One concrete achievement story that connects their background to the team's actual work. **If `profile.career_notes_md` is populated, use it as the primary source** — find the story block whose "Cover letter angle" best matches what this company is hiring for, and write from the full context (situation, decision, outcome) it contains. Do not just restate the resume bullet; write the version of the story that has room for the decision made and why it mattered. If career_notes is empty, draw from the resume but go deeper than the bullet — add context, name the decision, describe the outcome in a way the resume doesn't have space for.
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
4. PATCH `$BASE_URL/api/jobs/<job-id>` (see the PATCH table in the API contract; omitted fields keep their current values):
   ```json
   {"fit_score": <n>, "verdict": "<v>", "positives": "<json>", "concerns": "<json>", "summary": "<s>"}
   ```
   The PATCH returns the updated job. Confirm `fit_score` and `verdict` came back as sent before reporting the new score.
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
