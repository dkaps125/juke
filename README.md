# Juke.ai

Juke is an AI DJ that sits on top of your streaming platform of choice.

## Config

Juke supports configurations for:
- Music sources
- LLM providers

Configuration should be set to environment variables or in a `.env` file. Options are as follows.

```
LLM_PROVIDER = "ollama" | "openrouter"
MUSIC_SOURCE = "spotify
MODEL_NAME = "" # any model name supported by the LLM provider
```

Additional configuration options can be set via environment variables and are described in their related sections below.

## LLM Providers

### Ollama

Ollama is a great tool for running models locally. Download [here](http://ollama.com).

A couple of models that seem pretty good:
- gemma3:12b
- ministral-3:8b (probably 14b too)
- gemma3n:e4b (for testing, doesn't seem to have enough broad music knowledge for production use)

Obviously bigger is better here; use whatever model works best for you on your hardware.

Ollama's endpoint can be customized by setting the `OLLAMA_HOST` env var in the form `<scheme>://<host>:<port>`.

### Openrouter

Openrouter exposes an extensive model catalog through one interface. Set `OPENROUTER_API_KEY` in your environment to authenticate.
