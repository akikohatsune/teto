# Teto 

Discord AI role-play built with **Go** + **NVIDIA NIM**. Fast, secure, and very fat.

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go)](https://go.dev/)
[![NVIDIA](https://img.shields.io/badge/NVIDIA-76B900?style=flat&logo=nvidia)](https://www.nvidia.com/)
[![Discord](https://img.shields.io/badge/Discord-5865F2?style=flat&logo=discord)](https://discord.com/)
[![MIT](https://img.shields.io/badge/License-MIT-yellow)](LICENSE)

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

License: MIT | By: [Akiko Hatsune](https://github.com/akikohatsune)
