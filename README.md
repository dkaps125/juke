# Juke.ai

Juke is an AI DJ that sits on top of your streaming platform of choice.

## Config

Juke supports configurations for:
- Music sources
- LLM providers

Configuration should be set to environment variables or in a `.env` file. Options are as follows.

```
LLM_PROVIDER = "ollama"
MUSIC_SOURCE = "spotify
MODEL_NAME = "" # any model name supported by the LLM provider
```