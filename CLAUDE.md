<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
at `docs/rewrite/` (01-current-system.md, 02-decisions.md, 03-open-questions.md,
04-target-architecture.md, 05-migration-plan.md).

Active feature plan: `specs/002-initiative-overview-api/plan.md`
<!-- SPECKIT END -->

## Frontend coding rules

When building any UI element, **always load the `uikit` skill first** (`.claude/skills/uikit/SKILL.md`) to check whether an existing component covers the need before writing any HTML or creating a new component.
