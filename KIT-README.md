# CORE Stage 1 kit

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
