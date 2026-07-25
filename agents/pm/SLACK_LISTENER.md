# Slack Listener - Quick Start

Depois que você configurou seu Slack App (ver `SLACK_SETUP.md`), você pode rodar o listener.

## ⚡ Quick Start

```bash
# Make sure .env está preenchido com os 3 tokens Slack
python slack_listener.py
```

Você deve ver:

```
============================================================
🤖 PM Agent - Slack Listener
============================================================
✅ Connected as: @pm-agent (U0123XXXXX)
🔗 Connecting to Slack via Socket Mode...
✅ Connected to Slack via Socket Mode

📡 Listening for app mentions...
   Try: @pm-agent What's the status?

============================================================
```

Agora o bot tá **online no seu Slack!** 🎉

## 📝 Como usar

### Via menção direta:
```
@pm-agent What's the current status of the project?
```

### Via thread:
```
@pm-agent Can you summarize the backlog?
```

### Direct message:
```
DM @pm-agent com qualquer pergunta
```

O bot vai processar com o LLM que você escolheu e responder em tempo real.

---

## 🛠️ Troubleshooting

### Bot não responde
- [ ] Verifica se `SLACK_BOT_TOKEN` está correto
- [ ] Verifica se `SLACK_APP_TOKEN` está correto
- [ ] Verifica se o bot tá instalado no workspace
- [ ] Verifica logs no console pra erros

### "Invalid token"
- Volta em https://api.slack.com/apps
- Seleciona seu app
- Regenera os tokens
- Coloca no `.env`

### Bot aparece "offline" no Slack
- Certifique-se de que `python slack_listener.py` tá rodando
- Se foi interrompido (Ctrl+C), reinicia

### Mensagens não chegam no Linear
- Verifica se `LINEAR_API_KEY` tá correto
- Verifica a query no `pm_agent.py`

---

## 🚀 Rodar em Background

Se quer que o bot fique sempre online:

### Option 1: tmux (local)
```bash
tmux new-session -d -s pm-agent 'cd agents/pm && python slack_listener.py'
```

Pra parar:
```bash
tmux kill-session -t pm-agent
```

### Option 2: systemd (Linux)
Criar arquivo `/etc/systemd/system/pm-agent.service`:

```ini
[Unit]
Description=PM Agent Slack Listener
After=network.target

[Service]
Type=simple
User=diego
WorkingDirectory=/home/diego/Documents/frame/agents/pm
Environment="PATH=/home/diego/Documents/frame/agents/pm/venv/bin"
ExecStart=/home/diego/Documents/frame/agents/pm/venv/bin/python slack_listener.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Depois:
```bash
sudo systemctl enable pm-agent
sudo systemctl start pm-agent
sudo systemctl status pm-agent
```

### Option 3: screen (macOS/Linux)
```bash
screen -S pm-agent -d -m python slack_listener.py
```

---

## 📊 Monitorar

Enquanto tá rodando, o bot mostra:

```
📨 Message: What's the status?
🤔 PM Agent thinking...
✅ Responded in thread
```

Pra ver só os erros:
```bash
python slack_listener.py 2>&1 | grep "❌"
```

---

## ⏸️ Parar o Listener

```bash
Ctrl+C
```

Quando parar, o bot fica "offline" no Slack (ninguém pode usar).

---

## Próximos passos

- [ ] Rodar `slack_listener.py` pra testar no Slack
- [ ] Fazer algumas perguntas pro bot
- [ ] Deixar rodando 24/7 (usar tmux/systemd)
- [ ] Criar mais agentes (Dev Agent, QA Agent, etc)
