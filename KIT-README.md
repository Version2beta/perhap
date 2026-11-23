# CORE Stage 1 / pure REV Stage 2 kit

Current overlay: Publication Stage 3B completion. Binding docs/rddd/RECOVERY-SPEC.md.
Add retained planning/recoverable slots; unchanged SQLite format, dependencies and judge.
Historical overlay: Publication Stage 3B1 finite authorized in-process sessions.
Contract/map: docs/rddd/SESSION-SPEC.md. Bounded split: cursor/resume/resnapshot
and trust/expiry persistence remain 3B2. No new dependencies or wire formats.
Historical overlay: Publication Stage 3A private view journal. Binding contract:
docs/rddd/VIEWJOURNAL-SPEC.md. No new dependency; existing SQLite CGO toolchain.
New P3ACUT/P3ASTATE/P3ARECOVER/P3AALL tasks retain prior stages as regressions.
No external observations/cursors/sessions or runtime authority are implemented.
Historical overlay: Publication Stage 2 local durable delivery. Binding contract:
docs/rddd/DELIVERY-SPEC.md; new IDs D2STORE/D2RUN/D2CRASH/D2ALL. No dependency
changes. Confirmations and accepting sink are separate exclusive SQLite files.
The Stage 1 ledger is protected unchanged. Earlier overlay contract and source pin:
docs/rddd/LEDGER-SPEC.md; driver/durability/API: LEDGER-DESIGN.md. CGO and a C
compiler required for pinned mattn/go-sqlite3 v1.14.32. New task IDs LGPSTATE,
LGPDURABLE, LGPALL do not reuse earlier attempt counters. Preserve ignored logs.
Historical pure REV scope remains docs/rddd/REV-SPEC.md.
Earlier CORE tasks remain in tasks.json, protected and unchanged. R6 tasks use
new IDs; existing ignored .loop/ artifacts are retained. No reference code shipped.

Already applied on rddd-foundation; preserve existing repository files.
Prerequisites: Go per go.mod, Node/npm, jq, bash, timeout or gtimeout, configured Pi.
Frontend dependencies: cd demo/web && npm ci --ignore-scripts --no-audit --no-fund.

Verify: scripts/loop.sh --check (expected failures while implementation is stubbed).
After verified baseline commit, start from clean tree:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh

Exact unchanged skill defaults:
- pi --provider anthropic --model claude-haiku-5-5 --thinking high --print --no-session
- pi --provider openai-codex --model gpt-6.1-sol --thinking high --print --no-session
- pi --provider openai-codex --model gpt-6.1-sol --thinking max --print --no-session

Local judged commits stay on this branch. .loop/ contains logs and prompts;
STOP names a task exhausting attempts. No push. The private reference is outside
the kit and must never be consulted by the coder. See docs/rddd for evidence.
