# PM Agent - Project Manager

A minimal PM agent that:
- Accesses Linear for project status
- Responds to questions via Slack
- Supports Claude (Anthropic), GPT (OpenAI), or Gemini (Google) for reasoning

## Setup

### 1. Install dependencies
```bash
cd agents/pm
pip install -r requirements.txt
```

### 2. Get API keys

**Choose an LLM Provider:**

**Claude (Anthropic)** - Recommended
- Go to https://console.anthropic.com/
- Create an API key
- Set `LLM_PROVIDER=anthropic` in `.env`

**GPT (OpenAI)**
- Go to https://platform.openai.com/api/keys
- Create an API key
- Set `LLM_PROVIDER=openai` in `.env`

**Gemini (Google)**
- Go to https://aistudio.google.com/apikey
- Create an API key
- Set `LLM_PROVIDER=google` in `.env`

**Linear API:**
- Go to Linear workspace settings → API → Create personal API key
- Copy the key

**Slack:**
- Create a Slack App: https://api.slack.com/apps
- Enable Socket Mode (Settings → Socket Mode)
- Create an app token and bot token
- Add permissions: `chat:write`, `app_mentions:read`, `message.app_mentions:read`
- Install app to your workspace

### 3. Configure `.env`
```bash
cp .env.example .env
# Edit .env with your keys
```

### 4. Test the agent
```bash
python pm_agent.py
```

Should output:
```
PM Agent initialized. Testing...
1. Fetching issues from Linear...
   Found X issues
2. Testing message processing...
   Response: ...
```

## Running the Slack bot

(Next phase: event listener that responds to @pm-agent mentions)

```bash
python slack_listener.py
```

## Next steps

- [ ] Slack event listener (respond to mentions)
- [ ] More Linear queries (backlog, in progress, done)
- [ ] Better tool calling with Gemini
- [ ] Deploy to serverless (Cloud Functions)
