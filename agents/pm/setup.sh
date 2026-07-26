#!/bin/bash

echo "🤖 PM Agent Setup"
echo "================="
echo ""

# Check Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 not found. Please install Python 3.10+"
    exit 1
fi

echo "✅ Python found: $(python3 --version)"
echo ""

# Create venv
echo "📦 Creating virtual environment..."
python3 -m venv venv
source venv/bin/activate

echo "📥 Installing dependencies..."
pip install -q -r requirements.txt

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Copy .env.example to .env"
echo "2. Fill in your API keys:"
echo "   - GOOGLE_API_KEY (from aistudio.google.com)"
echo "   - LINEAR_API_KEY (from Linear workspace)"
echo "   - SLACK_BOT_TOKEN (from Slack app)"
echo ""
echo "3. Test the agent:"
echo "   source venv/bin/activate"
echo "   python pm_agent.py"
echo ""
