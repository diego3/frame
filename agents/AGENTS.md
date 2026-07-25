# Agent Team - GoEngine

Autonomous agents that work on the game engine while you review on Slack.

## Architecture

```
┌─────────────┐
│   Linear    │  Project issues, backlog, status
└──────┬──────┘
       │
       ├─────────────────┐
       │                 │
   ┌───▼────┐       ┌────▼────┐
   │  PM    │       │ Dev/CI  │  (future)
   │ Agent  │       │ Agents  │
   └───┬────┘       └────┬────┘
       │                 │
       └────────┬────────┘
                │
          ┌─────▼──────┐
          │   Slack    │  Your review interface
          └────────────┘
```

## Current Agents

### PM Agent (v0.1)
**Role**: Project Manager  
**Tech**: Google Gemini 2.0 Flash + Linear API  
**Status**: Setup phase

**Capabilities**:
- Fetch issues from Linear
- Summarize project status
- Answer questions about backlog, priorities
- Respond via Slack

**Location**: `agents/pm/`

## Future Agents

- **Dev Agent**: Implement features, write code, open PRs
- **QA Agent**: Run tests, find regressions
- **Doc Agent**: Generate docs, keep README updated

## Interaction Model

```
You (Slack)  →  "What's our status?"
    ↓
PM Agent  →  Query Linear  →  Gemini thinking  →  Response
    ↓
Slack  →  "3 issues in progress, 8 backlog..."
```

## Dev Notes

- Minimal setup: Just the essentials to test agent ↔ Linear ↔ Slack flow
- Using Google SDK for variety (not Claude API)
- Plan to move to production/serverless after v0.1 validation
