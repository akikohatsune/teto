# Teto 

Discord AI role-play built with **Go** + **NVIDIA NIM**. Fast, secure, and very fat.

## Features

-  **NVIDIA NIM** - Fast processing + image analysis
-  **Memory Isolation** - Each user has isolated chat history
-  **KomiFilter** - Protects against prompt injection, jailbreak
-  **Lightweight** - Go, low RAM, fast

## Installation

### Requirements
- Go 1.20+
- [NVIDIA API Key](https://build.nvidia.com/)
- [Discord Bot Token](https://discord.com/developers/applications)

### Setup

```bash
# Clone
git clone https://github.com/akikohatsune/teto.git
cd teto

# Install dependencies
go mod tidy

# Setup .env (auto-generated from .env.example)
# Fill in: NVIDIA_API_KEY, DISCORD_TOKEN, OWNER_USER_ID

# Run
go run main.go
```

## Commands

### Chat
- `@Teto <message>` - Chat directly
- `!chat <message>` - Chat via command (supports images)
- `!ask <message>` - Alias for !chat

### Admin
- `!clearmemo` - Clear memory for all users
- `!ban @user <reason>` - Ban user
- `!removeban @user` - Remove ban
- `!replayTeto ls` - View chat logs
- `!replayTeto <id>` - View chat details

### System
- `!provider` - View model info
- `!terminated on|off` - Toggle maintenance mode

## Security

**KomiFilter** protects through 3 layers:
1. Block prompt injection
2. Prevent system prompt leaks
3. Auto-check responses

## Database

Uses SQLite:
- `chat_memory.db` - Chat history (isolated per user)
- `ban_control.db` - Ban list
- `callnames.db` - Custom nicknames

---
## Note
Teto can be used through DMs or added to a server (recommended to keep it in a separate channel).
- [Link add bot](https://discord.com/oauth2/authorize?client_id=1095321922537005219)

I’ll keep it online as long as possible and make it as stable as I can.


---

License: MIT | By: [akikohatsune](https://github.com/akikohatsune)
