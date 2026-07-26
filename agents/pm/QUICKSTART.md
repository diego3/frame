# Quick Start - PM Agent

Get the PM agent running in 5 minutes.

## 1️⃣ Get API Keys

**Pick an LLM Provider:**

**Option A: Claude (Anthropic)** 🎯 Recommended
```
https://console.anthropic.com/
```
→ Copy API Key → Add to .env as `ANTHROPIC_API_KEY`

**Option B: GPT (OpenAI)**
```
https://platform.openai.com/api/keys
```
→ Copy API Key → Add to .env as `OPENAI_API_KEY`

**Option C: Gemini (Google)**
```
https://aistudio.google.com/apikey
```
→ Copy API Key → Add to .env as `GOOGLE_API_KEY`

**Linear:**
```
Linear App → Settings → Developer → Create Personal API Key
```

**Slack:**
```
https://api.slack.com/apps
```
→ Create New App → Select "From scratch"  
→ Settings → Socket Mode → Create App Token (read, write)  
→ Install App → Copy Bot User OAuth Token

## 2️⃣ Install & Configure

```bash
./setup.sh
cp .env.example .env
```

Edit `.env`:
```env
LLM_PROVIDER=anthropic          # or openai, google
ANTHROPIC_API_KEY=sk-...        # your chosen provider key
LINEAR_API_KEY=lin_...
SLACK_BOT_TOKEN=xoxb-...
```

## 3️⃣ Test

```bash
source venv/bin/activate
python pm_agent.py
```

Should show:
```
============================================================
PM Agent Test
============================================================
Provider: ANTHROPIC
Model: claude-3-5-sonnet-20241022

1️⃣  Fetching issues from Linear...
   ✅ Found 8 issues

2️⃣  Testing message processing...
   ✅ Response: Here's the current status...
```

✅ **Done!** Swap providers anytime by changing `LLM_PROVIDER` in `.env`.

## Switch Providers

Just edit `.env` and change `LLM_PROVIDER`:
```env
LLM_PROVIDER=openai    # ← or anthropic, google
OPENAI_API_KEY=sk-...
```

Then run tests again. Same agent, different brain!

## Next: Slack Integration

When ready to connect to Slack, run:
```bash
python slack_listener.py
```
