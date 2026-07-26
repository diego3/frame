#!/usr/bin/env python3
"""
Slack listener for PM Agent using Socket Mode.
Follows the official Slack SDK pattern.
"""

import os
import re
from pathlib import Path
from dotenv import load_dotenv
from slack_sdk import WebClient
from slack_sdk.socket_mode import SocketModeClient
from slack_sdk.socket_mode.request import SocketModeRequest
from slack_sdk.socket_mode.response import SocketModeResponse
from pm_agent import process_message

# Load .env
env_path = Path(__file__).parent / ".env"
load_dotenv(dotenv_path=env_path)

SLACK_APP_TOKEN = os.getenv("SLACK_APP_TOKEN")
SLACK_BOT_TOKEN = os.getenv("SLACK_BOT_TOKEN")

if not SLACK_APP_TOKEN or not SLACK_BOT_TOKEN:
    print("❌ Missing Slack tokens in .env")
    exit(1)

# Initialize clients
web_client = WebClient(token=SLACK_BOT_TOKEN)
socket_client = SocketModeClient(
    app_token=SLACK_APP_TOKEN,
    web_client=web_client
)

def process_socket_mode_request(client: SocketModeClient, req: SocketModeRequest):
    """Process Socket Mode requests - CORRECT PATTERN."""
    print(f"\n📬 Request type: {req.type}")

    if req.type == "events_api":
        # Acknowledge the request immediately
        response = SocketModeResponse(envelope_id=req.envelope_id)
        client.send_socket_mode_response(response)

        # Handle the event
        event = req.payload.get("event", {})
        event_type = event.get("type")

        print(f"   Event: {event_type}")

        if event_type == "app_mention":
            print(f"🎉 APP MENTION!")

            # Extract message
            text = event.get("text", "")
            cleaned_text = re.sub(r"<@[A-Z0-9]+>", "", text).strip()
            if not cleaned_text:
                cleaned_text = "What's the status?"

            print(f"   Message: {cleaned_text[:80]}")

            try:
                print(f"   🤔 Processing...")
                response_text = process_message(cleaned_text)

                # Send response
                web_client.chat_postMessage(
                    channel=event["channel"],
                    text=response_text,
                    thread_ts=event.get("thread_ts") or event["ts"]
                )
                print(f"   ✅ Response sent!")
            except Exception as e:
                print(f"   ❌ Error: {e}")

def main():
    print("\n" + "="*60)
    print("🤖 PM Agent - Slack Listener")
    print("="*60)

    # Test auth
    try:
        auth = web_client.auth_test()
        print(f"✅ Connected as: @{auth['user']}")
    except Exception as e:
        print(f"❌ Auth failed: {e}")
        exit(1)

    # Register listener with CORRECT pattern
    socket_client.socket_mode_request_listeners.append(process_socket_mode_request)

    print(f"\n🔗 Connecting to Socket Mode...")
    socket_client.connect()
    print(f"✅ Connected to Socket Mode!")

    print(f"\n📡 Listening for @pm mentions...")
    print(f"   Try: @pm What's the status?")
    print(f"\n{'='*60}\n")

    try:
        from threading import Event
        Event().wait()
    except KeyboardInterrupt:
        print("\n👋 Shutting down...")
        socket_client.close()

if __name__ == "__main__":
    main()
