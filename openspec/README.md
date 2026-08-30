# openspec/

Spec-driven definition of `wobble`, in the [OpenSpec](https://github.com/Fission-AI/OpenSpec)
layout.

```
openspec/
  project.md                     project context, tech stack, conventions
  specs/
    execution-duration/spec.md   run length + hard max + watchdog
    cpu-load/spec.md             CPU consumption model + accuracy
    exit-outcome/spec.md         success/failure draw -> exit code
    cli/spec.md                  flags/env, validation, seeding, signals,
                                 logging, and the exit-code table
  changes/                       proposed deltas (one dir per change), empty now
```

## Reading a spec

Each `spec.md` is:

- `## Purpose` — one paragraph on what the capability covers.
- `## Requirements` — a list of `### Requirement:` entries. Each requirement is a
  single normative sentence (using **SHALL** / **SHALL NOT** / **MAY**) followed
  by one or more `#### Scenario:` blocks that make it testable via
  `WHEN` / `THEN` / `AND` bullets.

Every scenario is intended to map to an acceptance test.

## Making a change

1. Create `changes/<verb>-<slug>/proposal.md` (why + what changes) and
   `tasks.md` (implementation checklist).
2. Add `changes/<verb>-<slug>/specs/<capability>/spec.md` containing only the
   `## ADDED`, `## MODIFIED`, or `## REMOVED` requirement blocks.
3. On merge, fold those deltas into `specs/<capability>/spec.md`.
