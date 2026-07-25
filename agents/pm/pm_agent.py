#!/usr/bin/env python3
"""
PM Agent: Project Manager with support for Claude, GPT, or Gemini.
Connects to Linear API and Slack for interaction.
"""

import os
import json
import requests
from pathlib import Path
from dotenv import load_dotenv
from slack_sdk import WebClient
from slack_sdk.errors import SlackApiError
from typing import Optional

# Load .env from current directory
env_path = Path(__file__).parent / ".env"
load_dotenv(dotenv_path=env_path)

# Configuration
LLM_PROVIDER = os.getenv("LLM_PROVIDER", "anthropic").lower()  # anthropic, openai, google
LLM_MODEL = os.getenv("LLM_MODEL", "claude-opus-4-1-20250805")  # Model to use
LINEAR_API_KEY = os.getenv("LINEAR_API_KEY")
SLACK_BOT_TOKEN = os.getenv("SLACK_BOT_TOKEN")
LINEAR_TEAM_KEY = os.getenv("LINEAR_TEAM_KEY", "DIE")

# Provider-specific imports and setup
if LLM_PROVIDER == "anthropic":
    import anthropic
    api_key = os.getenv("ANTHROPIC_API_KEY")
    if not api_key:
        raise ValueError("ANTHROPIC_API_KEY not set in .env")
    client = anthropic.Anthropic(api_key=api_key)
    MODEL = LLM_MODEL
elif LLM_PROVIDER == "openai":
    import openai
    openai.api_key = os.getenv("OPENAI_API_KEY")
    client = openai.OpenAI(api_key=os.getenv("OPENAI_API_KEY"))
    MODEL = "gpt-4o"
elif LLM_PROVIDER == "google":
    import google.generativeai as genai
    genai.configure(api_key=os.getenv("GOOGLE_API_KEY"))
    MODEL = "gemini-2.0-flash"
else:
    raise ValueError(f"Unknown LLM_PROVIDER: {LLM_PROVIDER}")

slack_client = WebClient(token=SLACK_BOT_TOKEN)
print(f"🤖 PM Agent initialized with provider: {LLM_PROVIDER} ({MODEL})")

# Linear API base
LINEAR_API_URL = "https://api.linear.app/graphql"

# ===== Tools =====

def get_linear_issues(status=None):
    """Fetch issues from Linear."""
    query = """
    query {
      issues(first: 20) {
        nodes {
          id
          identifier
          title
          description
          state {
            name
          }
          priority
          estimate
          team {
            key
            name
          }
        }
      }
    }
    """
    headers = {"Authorization": LINEAR_API_KEY}
    try:
        resp = requests.post(LINEAR_API_URL, json={"query": query}, headers=headers)
        if resp.status_code == 200:
            data = resp.json()
            all_issues = data.get("data", {}).get("issues", {}).get("nodes", [])
            # Filter by team locally
            issues = [i for i in all_issues if i.get("team", {}).get("key") == LINEAR_TEAM_KEY]
            return issues
        else:
            print(f"      Linear API error: {resp.status_code} - {resp.text[:100]}")
            return []
    except Exception as e:
        print(f"      Exception: {e}")
        return []

def get_issue_summary():
    """Get a summary of current issues."""
    issues = get_linear_issues()
    if not issues:
        return "No issues found."

    summary = f"Found {len(issues)} issues:\n"
    for issue in issues[:5]:  # Show top 5
        state_name = issue.get('state', {}).get('name', 'Unknown')
        summary += f"- {issue['identifier']}: {issue['title']} ({state_name})\n"
    if len(issues) > 5:
        summary += f"... and {len(issues) - 5} more\n"
    return summary

def get_backlog_count():
    """Count backlog items."""
    issues = get_linear_issues()
    backlog = [i for i in issues if i['status']['name'].lower() == 'backlog']
    return len(backlog)

# ===== Agent Logic =====

def get_system_prompt() -> str:
    """Get PM system prompt."""
    return """You are a Project Manager for the GoEngine game engine project.
Your job is to:
- Keep track of issues in Linear
- Answer questions about project status
- Suggest priorities and next steps
- Be encouraging and concise
- Help coordinate the agent team

Answer in a friendly, professional tone. Be direct and actionable."""

def process_message(user_message: str) -> str:
    """Process a user message and return PM response.

    Supports Claude (Anthropic), GPT (OpenAI), and Gemini (Google) LLMs.
    """
    system_prompt = get_system_prompt()

    # Get context from Linear
    context = f"\n\nCurrent project status:\n{get_issue_summary()}"
    enhanced_message = f"{user_message}{context}"

    if LLM_PROVIDER == "anthropic":
        response = client.messages.create(
            model=MODEL,
            max_tokens=1024,
            system=system_prompt,
            messages=[{"role": "user", "content": enhanced_message}]
        )
        return response.content[0].text

    elif LLM_PROVIDER == "openai":
        response = client.chat.completions.create(
            model=MODEL,
            max_tokens=1024,
            system=system_prompt,
            messages=[{"role": "user", "content": enhanced_message}]
        )
        return response.choices[0].message.content

    elif LLM_PROVIDER == "google":
        import google.generativeai as genai
        model = genai.GenerativeModel(
            model_name=MODEL,
            system_instruction=system_prompt,
        )
        response = model.generate_content(enhanced_message)
        return response.text if response.text else "I couldn't generate a response."

    return "Provider not configured."

def send_slack_message(channel: str, text: str, thread_ts: str = None):
    """Send a message to Slack."""
    try:
        kwargs = {"channel": channel, "text": text}
        if thread_ts:
            kwargs["thread_ts"] = thread_ts
        result = slack_client.chat_postMessage(**kwargs)
        return result["ts"]
    except SlackApiError as e:
        print(f"Error sending message: {e}")
        return None

# ===== Main Entry Point =====

if __name__ == "__main__":
    print(f"\n{'='*60}")
    print(f"PM Agent Test")
    print(f"{'='*60}")
    print(f"Provider: {LLM_PROVIDER.upper()}")
    print(f"Model: {MODEL}")

    # Debug: Check env vars
    print(f"\n📋 Config Check:")
    print(f"   LINEAR_API_KEY: {LINEAR_API_KEY[:20] if LINEAR_API_KEY else 'NOT SET'}...")
    print(f"   LINEAR_TEAM_KEY: {LINEAR_TEAM_KEY}")

    # Test: Get issues
    print("\n1️⃣  Fetching issues from Linear...")
    try:
        issues = get_linear_issues()
        print(f"   ✅ Found {len(issues)} issues")
        if issues:
            for issue in issues[:2]:
                print(f"      - {issue['identifier']}: {issue['title']}")
    except Exception as e:
        print(f"   ❌ Error: {e}")

    # Test: Process a message
    print("\n2️⃣  Testing message processing...")
    try:
        response = process_message("What's the current status of the game engine project?")
        print(f"   ✅ Response:\n   {response[:300]}...")
    except Exception as e:
        print(f"   ❌ Error: {e}")

    print(f"\n{'='*60}")
    print("✅ Setup complete! Next: Configure Slack event listener.")
    print(f"{'='*60}\n")
