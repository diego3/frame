# LLM Provider Configuration

The PM Agent supports 3 LLM providers. Switch anytime by changing `LLM_PROVIDER` in `.env`.

## Anthropic (Claude)

**Recommended** - Best reasoning, most reliable.

```env
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...
```

Get key: https://console.anthropic.com/

Model used: `claude-3-5-sonnet-20241022`

## OpenAI (GPT)

**Fastest** - Good reasoning, widely used.

```env
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-proj-...
```

Get key: https://platform.openai.com/api/keys

Model used: `gpt-4o`

## Google (Gemini)

**Latest** - Good at tasks, experimental.

```env
LLM_PROVIDER=google
GOOGLE_API_KEY=AIzaSy...
```

Get key: https://aistudio.google.com/apikey

Model used: `gemini-2.0-flash`

---

## Switching Providers

1. Edit `.env`:
   ```bash
   LLM_PROVIDER=openai
   ```

2. Make sure you have the API key:
   ```bash
   OPENAI_API_KEY=sk-proj-...
   ```

3. Test immediately:
   ```bash
   python pm_agent.py
   ```

Same agent, different LLM. No code changes needed.

## Pricing

As of July 2024:

| Provider | Model | Input | Output |
|----------|-------|-------|--------|
| Anthropic | Claude 3.5 Sonnet | $3/1M tokens | $15/1M tokens |
| OpenAI | GPT-4o | $5/1M tokens | $15/1M tokens |
| Google | Gemini 2.0 Flash | $0.075/1M tokens | $0.30/1M tokens |

For low-volume testing, all three are practically free.
