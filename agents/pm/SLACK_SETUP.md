# Slack Setup - PM Agent

Guia completo pra conectar seu PM agent no Slack.

## 📋 Resumo do que você vai fazer

1. Criar um Slack App
2. Habilitar Socket Mode (conexão em tempo real)
3. Gerar tokens
4. Adicionar permissões
5. Instalar no seu workspace
6. Testar a conexão

**Tempo: ~5 minutos**

---

## Step 1: Criar um Slack App

1. Acesse: https://api.slack.com/apps
2. Clique em **"Create New App"**
3. Selecione **"From scratch"**
4. Preencha:
   - **App name**: `PM Agent` (ou o nome que quiser)
   - **Pick a workspace**: Selecione seu workspace pessoal

5. Clique em **"Create App"**

✅ Você será redirecionado pra página de configuração do app.

---

## Step 2: Habilitar Socket Mode

Socket Mode permite que o app ouça eventos do Slack em tempo real (sem precisa de webhook público).

1. Na barra lateral esquerda, procure por **"Socket Mode"**
2. Clique em **"Enable Socket Mode"**
3. Uma janela vai pedir confirmação, clique em **"Enable"**

✅ Socket Mode ativado.

---

## Step 3: Gerar os Tokens

### Token de Conexão (App Token)

Ainda na página de Socket Mode:

1. Vá em **"App-Level Tokens"** 
2. Clique em **"Generate Token and Scopes"**
3. Dê um nome: `socket-mode` (opcional)
4. Adicione a permissão: `connections:write`
5. Clique em **"Generate"**
6. **Copie o token** que começa com `xapp-1-...`
   - Este é seu **APP_TOKEN** (não o bot token!)

❌ **NÃO feche essa página ainda**, vamos usar ela denovo.

### Token do Bot (Bot User Token)

1. Na barra lateral, clique em **"OAuth & Permissions"**
2. Scroll até **"Scopes"** → **"Bot Token Scopes"**
3. Clique em **"Add an OAuth Scope"** e adicione:
   - `chat:write` (escrever mensagens)
   - `app_mentions:read` (ler menções)
   - `channels:read` (listar canais)
   - `groups:read` (listar DMs)

Depois de adicionar os scopes:

4. Scroll pra cima até **"OAuth Tokens for Your Workspace"**
5. Clique em **"Install to Workspace"** (se não tiver instalado)
6. Autorize o app
7. **Copie o Bot User OAuth Token** que começa com `xoxb-...`
   - Este é seu **SLACK_BOT_TOKEN**

---

## Step 4: Adicionar Signing Secret

O Signing Secret é pra verificar que as requisições vêm do Slack.

1. Na barra lateral, clique em **"Basic Information"**
2. Scroll até **"App Credentials"**
3. Você verá **"Signing Secret"**
4. Clique no ícone de olho ou **"Show"**
5. **Copie o Signing Secret** que começa com `8a1f7...`

---

## Step 5: Guardar os Tokens

Você agora tem 3 tokens. Coloca eles no seu `.env`:

```env
# Socket Mode - conexão em tempo real
SLACK_APP_TOKEN=xapp-1-...

# Bot - permissão pra falar no Slack
SLACK_BOT_TOKEN=xoxb-...

# Segurança - valida requisições
SLACK_SIGNING_SECRET=8a1f7...
```

Se criou o `.env` a partir de `.env.example`, já tem os campos, é só preencher.

---

## Step 6: Instalar o App no Seu Workspace

1. Volta em **"Basic Information"**
2. Scroll até **"Install your app"**
3. Clique em **"Install to Workspace"**
4. Escolha o canal onde o bot vai responder (padrão: todos)
5. Clique em **"Allow"**

✅ O app agora está instalado.

---

## Step 7: Testar a Conexão

Agora você pode testar se tudo tá funcionando:

```bash
# Certifique-se de que .env está preenchido
python slack_listener.py
```

Você deve ver algo como:
```
🤖 Slack listener starting...
✅ Connected to Slack via Socket Mode
Waiting for messages...
```

Aí você vai pro Slack e menciona o bot:

```
@PM Agent what's the status?
```

E o bot deveria responder! 🎉

---

## 🐛 Troubleshooting

### "Invalid token" error
- Verifica se copiou a chave correta
- Claude tokens começam com `sk-`
- App tokens começam com `xapp-`
- Bot tokens começam com `xoxb-`

### Bot não responde
- Verifica se o bot tá instalado no workspace
- Verifica as permissões em "OAuth & Permissions"
- Restart o `slack_listener.py`

### "Socket Mode not enabled"
- Volta em Socket Mode e ativa
- Gera um novo app token
- Coloca no `.env`

### Bot aparece offline
- Certifique-se de que `slack_listener.py` tá rodando
- Verifica os tokens no `.env`
- Check o console pra erros

---

## ✅ Checklist Final

- [ ] Slack App criado em https://api.slack.com/apps
- [ ] Socket Mode ativado
- [ ] `SLACK_APP_TOKEN` gerado (xapp-...)
- [ ] `SLACK_BOT_TOKEN` gerado (xoxb-...)
- [ ] `SLACK_SIGNING_SECRET` copiado
- [ ] Tokens colados no `.env`
- [ ] App instalado no seu workspace
- [ ] `python slack_listener.py` rodando
- [ ] Bot responde no Slack

Próximo passo: Criar o arquivo `slack_listener.py` que fica escutando por mensagens!
