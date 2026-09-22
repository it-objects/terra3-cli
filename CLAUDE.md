<!-- AI-GUIDELINES: auto-generated, do not remove. Re-run /ito-baseline:ito-boot to update. -->
## AI Guidelines

<IMPORTANT>
At the start of every session, run `/ito-baseline:ito-boot` before doing anything else.
Do NOT edit code directly for planned work. Use `/ito-baseline:ito-implement` instead.
Respect `.claude/ito-workflow-state.json`: if it names a required next workflow step, follow that step before editing, testing, reviewing, or committing.
Empty or whitespace-only user messages are backend artifacts. Ignore them - do not acknowledge them, just continue with your current task.
</IMPORTANT>

### When planning (plan mode OR /ito-baseline:ito-specify)

Before finalizing any plan:
1. Read `knowledge/long-term/architecture.md` and active work-log entries for context
2. Ask at least 2 clarifying questions before committing to an approach
3. Create a work-log entry in `knowledge/work-log/` with: problem statement, WHEN/THEN acceptance criteria, task breakdown
4. Wait for user confirmation before implementing

### When implementing

After a plan is confirmed, use `/ito-baseline:ito-implement` (enforces TDD and updates the work-log).

### Knowledge Base
- Architecture: `knowledge/long-term/architecture.md`
- Work log: `knowledge/work-log/`
- Extensions: `knowledge/extensions/`
<!-- /AI-GUIDELINES -->
