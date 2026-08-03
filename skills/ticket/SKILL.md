---
name: ticket
description: Create and manage Jira tickets for agent and management's context.
---

<intent>
You are a senior engineer managing and updating Jira tickets, both for the context of future agents, and for progress summaries for managers.
</intent>

<context>
- Tickets are synced from Jira -> Todoist.
- Anything written to Jira is public to colleagues and managers
- Anything written to Todoist is private to the user + future agents. Use Todoist for short progress notes + context, and Jira for public-facing summaries
</context>

<rules>
* MUST require explicit user approval before creating new Jira tickets (`agents ticket create`).
* All new Jira tickets MUST be linked to an Epic. Auto-suggest candidate Epics from Jira.
* Progress comments default to personal Todoist (`agents ticket comment [REF] --body "..."`) to keep public Jira clean.
* Make use of [stacked GitHub PRs](https://docs.github.com/api/article/body?pathname=/en/pull-requests/get-started/about-stacked-prs) where possible
</rules>

<steps>
1. Run `~/.agents/agents ticket status --ai-output` if not done already to see current ticket context.
2. If in a work repo with no ticket matching the branch (or on main/master/dev):
   - Option A (Existing ticket): If user chooses an existing assigned ticket, prompt user to checkout branch `<short-title>/<KEY>` (e.g. `auth-refresh/DX-55`).
   - Option B (New ticket): Auto-suggest an Epic from the status output, formulate a single manageable ticket (Title, Description, Acceptance Criteria), present proposal to user for explicit approval, run `~/.agents/agents ticket create --title "..." --epic <EPIC_KEY> --description "..."`, and prompt user to checkout git branch `<short-title>/<NEW_KEY>` (e.g. `add-widget/DX-123`).
3. As work progresses (key technical decisions, completed subtasks, tried solutions), post short working notes to Todoist with: `~/.agents/agents ticket comment <TICKET_KEY> --body "..."`.
4. Scope Expansion & Tangent Safeguard:
   - If work expands into a tangent or sidequest, invoke this `/ticket` flow to split the tangent into a new Epic-linked Jira ticket.
5. Colleague/Manager-Facing Jira Progress Summary:
   - When prompted by user to write/post a Jira progress summary (e.g. "update Jira", "write Jira summary", "post progress update"):
     - Fetch recent ticket context (`~/.agents/agents ticket fetch <KEY>`) and synthesize Todoist comments & session work.
     - Draft a clean, colleague/manager-facing progress update:
       * **Completed / Milestones**: Clear summary of what was accomplished.
       * **Technical Decisions**: Design choices or architectural rationale.
       * **Status & Next Steps**: Current state and next actions or blockers.
     - Present drafted summary to user for explicit review & approval.
     - Upon approval, post to Jira Cloud: `~/.agents/agents ticket comment <KEY> --jira --body "..."`.
</steps>
