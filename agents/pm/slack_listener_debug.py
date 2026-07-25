#!/usr/bin/env python3
"""
Debug version of Slack listener with detailed logging.
"""

import os
import re
import json
from pathlib import Path
from dotenv import load_dotenv
from slack_sdk import WebClient
from slack_sdk.socket_mode import SocketModeClient
from slack_sdk.socket_mode.response import SocketModeResponse

# Load .env
env_path = Path(__file__).parent / ".env"
load_dotenv(dotenv_path=env_path)

SLACK_APP_TOKEN = os.getenv("SLACK_APP_TOKEN")
SLACK_BOT_TOKEN = os.getenv("SLACK_BOT_TOKEN")

print(f"App token: {SLACK_APP_TOKEN[:30]}..." if SLACK_APP_TOKEN else "NOT SET")
print(f"Bot token: {SLACK_BOT_TOKEN[:30]}..." if SLACK_BOT_TOKEN else "NOT SET")

if not SLACK_APP_TOKEN or not SLACK_BOT_TOKEN:
    print("❌ Missing tokens")
    exit(1)

web_client = WebClient(token=SLACK_BOT_TOKEN)
socket_client = SocketModeClient(app_token=SLACK_APP_TOKEN, web_client=web_client)

print("\n🤖 Starting Slack Listener (Debug Mode)")

# Test auth
try:
    auth = web_client.auth_test()
    print(f"✅ Auth OK: @{auth['user']}")
    print(f"   Team: {auth['team']}")
    print(f"   User ID: {auth['user_id']}")
except Exception as e:
    print(f"❌ Auth failed: {e}")
    exit(1)

def handle_request(ack, body):
    """Debug request handler."""
    print(f"\n📬 Got request!")
    print(f"   Type: {body.get('type')}")

    ack()

    if body.get("type") == "events_api":
        print(f"✅ Events API")
        event = body.get("event", {})
        print(f"   Event type: {event.get('type')}")
        print(f"   Full event: {json.dumps(event, indent=2)}")

        if event.get("type") == "app_mention":
            print(f"🎉 APP MENTION DETECTED!")
            channel = event.get("channel")
            text = event.get("text", "")
            user = event.get("user")
            print(f"   Channel: {channel}")
            print(f"   User: {user}")
            print(f"   Text: {text}")

            # Try to send a test response
            try:
                print(f"   Sending test response...")
                web_client.chat_postMessage(
                    channel=channel,
                    text="✅ Got your message! PM Agent is working!",
                    thread_ts=event.get("thread_ts") or event["ts"]
                )
                print(f"   ✅ Message sent!")
            except Exception as e:
                print(f"   ❌ Error sending: {e}")
    else:
        print(f"Other event type: {body.get('type')}")

print("\n🔗 Connecting to Socket Mode...")
try:
    socket_client.socket_mode_request_listeners.append(handle_request)
    socket_client.connect()
    print("✅ Connected to Socket Mode!")
    print("\n📡 Waiting for events... (Ctrl+C to stop)")
    print("   Try mentioning @pm in Slack\n")

    from threading import Event
    Event().wait()
except Exception as e:
    print(f"❌ Connection error: {e}")
    import traceback
    traceback.print_exc()
