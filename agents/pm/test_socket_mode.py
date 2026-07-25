#!/usr/bin/env python3
"""
Test Socket Mode connection directly.
"""

import os
from pathlib import Path
from dotenv import load_dotenv
from slack_sdk.socket_mode import SocketModeClient
from slack_sdk import WebClient

# Load .env
env_path = Path(__file__).parent / ".env"
load_dotenv(dotenv_path=env_path)

SLACK_APP_TOKEN = os.getenv("SLACK_APP_TOKEN")
SLACK_BOT_TOKEN = os.getenv("SLACK_BOT_TOKEN")

print(f"🔍 Testing Socket Mode Connection\n")
print(f"App Token: {SLACK_APP_TOKEN[:40]}...")
print(f"Bot Token: {SLACK_BOT_TOKEN[:40]}...")

# Test Web Client
print(f"\n1️⃣  Testing Web Client (auth)...")
try:
    web = WebClient(token=SLACK_BOT_TOKEN)
    auth = web.auth_test()
    print(f"   ✅ Auth OK: @{auth['user']}")
except Exception as e:
    print(f"   ❌ Auth failed: {e}")
    exit(1)

# Test Socket Mode
print(f"\n2️⃣  Testing Socket Mode...")
try:
    socket = SocketModeClient(app_token=SLACK_APP_TOKEN, web_client=web)

    print(f"   Connecting...")
    socket.connect()
    print(f"   ✅ Connected to Socket Mode!")
    print(f"   Status: {socket.is_connected()}")

    # Test send message
    print(f"\n3️⃣  Testing message send...")
    try:
        result = web.chat_postMessage(
            channel="#general",
            text="🤖 Bot test message"
        )
        print(f"   ✅ Message posted: {result['ts']}")
    except Exception as e:
        print(f"   ⚠️  Could not post to #general: {e}")
        print(f"      (Channel might not exist or bot has no access)")

    print(f"\n4️⃣  Waiting for events...")
    print(f"   Try mentioning @pm in ANY channel")
    print(f"   (Will print any event received)\n")

    # Simple event handler
    event_count = [0]
    def on_any_event(ack, body):
        event_count[0] += 1
        print(f"   📨 EVENT #{event_count[0]}: {body.get('type')}")
        if body.get("type") == "events_api":
            event = body.get("event", {})
            print(f"      └─ Event type: {event.get('type')}")
            if event.get("type") == "app_mention":
                print(f"      └─ ✅ APP MENTION!")
        ack()

    socket.socket_mode_request_listeners.append(on_any_event)

    # Wait
    from threading import Event
    Event().wait()

except Exception as e:
    print(f"   ❌ Socket Mode failed: {e}")
    import traceback
    traceback.print_exc()
