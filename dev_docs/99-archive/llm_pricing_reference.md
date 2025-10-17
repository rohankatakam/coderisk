# LLM & Embedding Pricing Reference (OpenAI + Gemini)

This reference aggregates raw pricing data for OpenAI and Google Gemini language, embedding, image, audio, and tooling services. It also documents how Cognee can interface with these providers for LLM and embedding workloads, including configuration and rate limiting hooks.

---

## Table of Contents
- [OpenAI Pricing](#openai-pricing)
  - [Text Tokens (per 1M tokens)](#text-tokens-per-1m-tokens)
    - [Batch Tier](#batch-tier)
    - [Flex Tier](#flex-tier)
    - [Standard Tier](#standard-tier)
    - [Priority Tier](#priority-tier)
  - [Image Tokens](#image-tokens)
  - [Audio Tokens](#audio-tokens)
  - [Fine-Tuning Prices](#fine-tuning-prices)
  - [Built-in Tool Charges](#built-in-tool-charges)
  - [Transcription & Speech Generation](#transcription--speech-generation)
  - [Image Generation (Per Image)](#image-generation-per-image)
  - [Embeddings](#embeddings)
  - [Moderation](#moderation)
  - [Legacy Models](#legacy-models)
- [Gemini Pricing](#gemini-pricing)
  - [Gemini 2.5 Series](#gemini-25-series)
  - [Gemini 2.0 Series](#gemini-20-series)
  - [Imagen](#imagen)
  - [Veo (Video Generation)](#veo-video-generation)
  - [Gemini Embedding](#gemini-embedding)
  - [Gemma](#gemma)
  - [Gemini 1.5 Series](#gemini-15-series)
- [Cognee Integration Notes](#cognee-integration-notes)
  - [LLM Provider Configuration](#llm-provider-configuration)
  - [Embedding Provider Configuration](#embedding-provider-configuration)
  - [Rate Limiting & Testing Controls](#rate-limiting--testing-controls)
  - [Operational Considerations](#operational-considerations)

---

## OpenAI Pricing

### Text Tokens (per 1M tokens)

#### Batch Tier
| Model | Input | Cached input | Output |
| --- | --- | --- | --- |
| gpt-5 | $0.625 | $0.0625 | $5.00 |
| gpt-5-mini | $0.125 | $0.0125 | $1.00 |
| gpt-5-nano | $0.025 | $0.0025 | $0.20 |
| gpt-4.1 | $1.00 | - | $4.00 |
| gpt-4.1-mini | $0.20 | - | $0.80 |
| gpt-4.1-nano | $0.05 | - | $0.20 |
| gpt-4o | $1.25 | - | $5.00 |
| gpt-4o-2024-05-13 | $2.50 | - | $7.50 |
| gpt-4o-mini | $0.075 | - | $0.30 |
| o1 | $7.50 | - | $30.00 |
| o1-pro | $75.00 | - | $300.00 |
| o3-pro | $10.00 | - | $40.00 |
| o3 | $1.00 | - | $4.00 |
| o3-deep-research | $5.00 | - | $20.00 |
| o4-mini | $0.55 | - | $2.20 |
| o4-mini-deep-research | $1.00 | - | $4.00 |
| o3-mini | $0.55 | - | $2.20 |
| o1-mini | $0.55 | - | $2.20 |
| computer-use-preview | $1.50 | - | $6.00 |

#### Flex Tier
| Model | Input | Cached input | Output |
| --- | --- | --- | --- |
| gpt-5 | $0.625 | $0.0625 | $5.00 |
| gpt-5-mini | $0.125 | $0.0125 | $1.00 |
| gpt-5-nano | $0.025 | $0.0025 | $0.20 |
| o3 | $1.00 | $0.25 | $4.00 |
| o4-mini | $0.55 | $0.138 | $2.20 |

#### Standard Tier
| Model | Input | Cached input | Output |
| --- | --- | --- | --- |
| gpt-5 | $1.25 | $0.125 | $10.00 |
| gpt-5-mini | $0.25 | $0.025 | $2.00 |
| gpt-5-nano | $0.05 | $0.005 | $0.40 |
| gpt-5-chat-latest | $1.25 | $0.125 | $10.00 |
| gpt-4.1 | $2.00 | $0.50 | $8.00 |
| gpt-4.1-mini | $0.40 | $0.10 | $1.60 |
| gpt-4.1-nano | $0.10 | $0.025 | $0.40 |
| gpt-4o | $2.50 | $1.25 | $10.00 |
| gpt-4o-2024-05-13 | $5.00 | - | $15.00 |
| gpt-4o-mini | $0.15 | $0.075 | $0.60 |
| gpt-realtime | $4.00 | $0.40 | $16.00 |
| gpt-4o-realtime-preview | $5.00 | $2.50 | $20.00 |
| gpt-4o-mini-realtime-preview | $0.60 | $0.30 | $2.40 |
| gpt-audio | $2.50 | - | $10.00 |
| gpt-4o-audio-preview | $2.50 | - | $10.00 |
| gpt-4o-mini-audio-preview | $0.15 | - | $0.60 |
| o1 | $15.00 | $7.50 | $60.00 |
| o1-pro | $150.00 | - | $600.00 |
| o3-pro | $20.00 | - | $80.00 |
| o3 | $2.00 | $0.50 | $8.00 |
| o3-deep-research | $10.00 | $2.50 | $40.00 |
| o4-mini | $1.10 | $0.275 | $4.40 |
| o4-mini-deep-research | $2.00 | $0.50 | $8.00 |
| o3-mini | $1.10 | $0.55 | $4.40 |
| o1-mini | $1.10 | $0.55 | $4.40 |
| codex-mini-latest | $1.50 | $0.375 | $6.00 |
| gpt-4o-mini-search-preview | $0.15 | - | $0.60 |
| gpt-4o-search-preview | $2.50 | - | $10.00 |
| computer-use-preview | $3.00 | - | $12.00 |
| gpt-image-1 | $5.00 | $1.25 | - |

#### Priority Tier
| Model | Input | Cached input | Output |
| --- | --- | --- | --- |
| gpt-5 | $2.50 | $0.25 | $20.00 |
| gpt-5-mini | $0.45 | $0.045 | $3.60 |
| gpt-4.1 | $3.50 | $0.875 | $14.00 |
| gpt-4.1-mini | $0.70 | $0.175 | $2.80 |
| gpt-4.1-nano | $0.20 | $0.05 | $0.80 |
| gpt-4o | $4.25 | $2.125 | $17.00 |
| gpt-4o-2024-05-13 | $8.75 | - | $26.25 |
| gpt-4o-mini | $0.25 | $0.125 | $1.00 |
| o3 | $3.50 | $0.875 | $14.00 |
| o4-mini | $2.00 | $0.50 | $8.00 |

> **Tier Guidance:** Priority tier boosts throughput; Flex reduces cost at higher latency; Batch provides discounted offline processing.

### Image Tokens
Prices per 1M tokens.

| Model | Input | Cached input | Output |
| --- | --- | --- | --- |
| gpt-image-1 | $10.00 | $2.50 | $40.00 |
| gpt-realtime | $5.00 | $0.40 | - |

### Audio Tokens
Prices per 1M tokens.

| Model | Input | Cached input | Output |
| --- | --- | --- | --- |
| gpt-realtime | $32.00 | $0.40 | $64.00 |
| gpt-4o-realtime-preview | $40.00 | $2.50 | $80.00 |
| gpt-4o-mini-realtime-preview | $10.00 | $0.30 | $20.00 |
| gpt-audio | $40.00 | - | $80.00 |
| gpt-4o-audio-preview | $40.00 | - | $80.00 |
| gpt-4o-mini-audio-preview | $10.00 | - | $20.00 |

### Fine-Tuning Prices
Prices per 1M tokens (training price per hour noted where provided).

#### Batch
| Model | Training | Input | Cached input | Output |
| --- | --- | --- | --- | --- |
| o4-mini-2025-04-16 | $100.00 / hour | $2.00 | $0.50 | $8.00 |
| o4-mini-2025-04-16 with data sharing | $100.00 / hour | $1.00 | $0.25 | $4.00 |
| gpt-4.1-2025-04-14 | $25.00 | $1.50 | $0.50 | $6.00 |
| gpt-4.1-mini-2025-04-14 | $5.00 | $0.40 | $0.10 | $1.60 |
| gpt-4.1-nano-2025-04-14 | $1.50 | $0.10 | $0.025 | $0.40 |
| gpt-4o-2024-08-06 | $25.00 | $2.225 | $0.90 | $12.50 |
| gpt-4o-mini-2024-07-18 | $3.00 | $0.15 | $0.075 | $0.60 |
| gpt-3.5-turbo | $8.00 | $1.50 | - | $3.00 |
| davinci-002 | $6.00 | $6.00 | - | $6.00 |
| babbage-002 | $0.40 | $0.80 | - | $0.90 |

#### Standard
| Model | Training | Input | Cached input | Output |
| --- | --- | --- | --- | --- |
| o4-mini-2025-04-16 | $100.00 / hour | $4.00 | $1.00 | $16.00 |
| o4-mini-2025-04-16 with data sharing | $100.00 / hour | $2.00 | $0.50 | $8.00 |
| gpt-4.1-2025-04-14 | $25.00 | $3.00 | $0.75 | $12.00 |
| gpt-4.1-mini-2025-04-14 | $5.00 | $0.80 | $0.20 | $3.20 |
| gpt-4.1-nano-2025-04-14 | $1.50 | $0.20 | $0.05 | $0.80 |
| gpt-4o-2024-08-06 | $25.00 | $3.75 | $1.875 | $15.00 |
| gpt-4o-mini-2024-07-18 | $3.00 | $0.30 | $0.15 | $1.20 |
| gpt-3.5-turbo | $8.00 | $3.00 | - | $6.00 |
| davinci-002 | $6.00 | $12.00 | - | $12.00 |
| babbage-002 | $0.40 | $1.60 | - | $1.60 |

> Tokens used for model grading in reinforcement fine-tuning are billed at the model’s per-token rate. Data sharing discounts apply when enabled.

### Built-in Tool Charges
| Tool | Cost |
| --- | --- |
| Code Interpreter | $0.03 / container |
| File search storage | $0.10 / GB per day (1 GB free) |
| File search tool call (Responses API only) | $2.50 / 1k calls |
| Web search preview (gpt-4o, gpt-4.1, gpt-4o-mini, gpt-4.1-mini) | $25.00 / 1k calls |
| Web search preview (gpt-5, o-series) | $10.00 / 1k calls |
| Web search (all models) | $10.00 / 1k calls |

> Search content tokens are billed at the model’s text token rates unless otherwise noted.

### Transcription & Speech Generation
Prices per 1M tokens unless noted.

#### Text Tokens
| Model | Input | Output | Estimated cost |
| --- | --- | --- | --- |
| gpt-4o-mini-tts | $0.60 | - | $0.015 / minute |
| gpt-4o-transcribe | $2.50 | $10.00 | $0.006 / minute |
| gpt-4o-mini-transcribe | $1.25 | $5.00 | $0.003 / minute |

#### Audio Tokens
| Model | Input | Output | Estimated cost |
| --- | --- | --- | --- |
| gpt-4o-mini-tts | - | $12.00 | $0.015 / minute |
| gpt-4o-transcribe | $6.00 | - | $0.006 / minute |
| gpt-4o-mini-transcribe | $3.00 | - | $0.003 / minute |

#### Other Models
| Model | Use case | Cost |
| --- | --- | --- |
| Whisper | Transcription | $0.006 / minute |
| TTS | Speech generation | $15.00 / 1M characters |
| TTS HD | Speech generation | $30.00 / 1M characters |

### Image Generation (Per Image)
| Model | Quality | 1024×1024 | 1024×1536 | 1536×1024 |
| --- | --- | --- | --- | --- |
| GPT Image 1 | Low | $0.011 | $0.016 | $0.016 |
|  | Medium | $0.042 | $0.063 | $0.063 |
|  | High | $0.167 | $0.25 | $0.25 |
| DALL·E 3 | Standard | $0.04 | $0.08 | $0.08 |
|  | HD | $0.08 | $0.12 | $0.12 |
| DALL·E 2 | Standard | $0.016 | $0.018 | $0.02 |

### Embeddings
| Model | Cost (per 1M tokens) | Batch cost |
| --- | --- | --- |
| text-embedding-3-small | $0.02 | $0.01 |
| text-embedding-3-large | $0.13 | $0.065 |
| text-embedding-ada-002 | $0.10 | $0.05 |

### Moderation
OpenAI’s omni-moderation models are offered free of charge.

### Legacy Models

#### Batch
| Model | Input | Output |
| --- | --- | --- |
| gpt-4-turbo-2024-04-09 | $5.00 | $15.00 |
| gpt-4-0125-preview | $5.00 | $15.00 |
| gpt-4-1106-preview | $5.00 | $15.00 |
| gpt-4-1106-vision-preview | $5.00 | $15.00 |
| gpt-4-0613 | $15.00 | $30.00 |
| gpt-4-0314 | $15.00 | $30.00 |
| gpt-4-32k | $30.00 | $60.00 |
| gpt-3.5-turbo-0125 | $0.25 | $0.75 |
| gpt-3.5-turbo-1106 | $1.00 | $2.00 |
| gpt-3.5-turbo-0613 | $1.50 | $2.00 |
| gpt-3.5-0301 | $1.50 | $2.00 |
| gpt-3.5-turbo-16k-0613 | $1.50 | $2.00 |
| davinci-002 | $1.00 | $1.00 |
| babbage-002 | $0.20 | $0.20 |

#### Standard
| Model | Input | Output |
| --- | --- | --- |
| chatgpt-4o-latest | $5.00 | $15.00 |
| gpt-4-turbo-2024-04-09 | $10.00 | $30.00 |
| gpt-4-0125-preview | $10.00 | $30.00 |
| gpt-4-1106-preview | $10.00 | $30.00 |
| gpt-4-1106-vision-preview | $10.00 | $30.00 |
| gpt-4-0613 | $30.00 | $60.00 |
| gpt-4-0314 | $30.00 | $60.00 |
| gpt-4-32k | $60.00 | $120.00 |
| gpt-3.5-turbo | $0.50 | $1.50 |
| gpt-3.5-turbo-0125 | $0.50 | $1.50 |
| gpt-3.5-turbo-1106 | $1.00 | $2.00 |
| gpt-3.5-turbo-0613 | $1.50 | $2.00 |
| gpt-3.5-0301 | $1.50 | $2.00 |
| gpt-3.5-turbo-instruct | $1.50 | $2.00 |
| gpt-3.5-turbo-16k-0613 | $3.00 | $4.00 |
| davinci-002 | $2.00 | $2.00 |
| babbage-002 | $0.40 | $0.40 |

---

## Gemini Pricing

> **Tier Notes:** Gemini offers a free tier (lower rate limits, default data sharing), a paid tier with higher limits and disabled product-improvement usage, and discounted Batch pricing (50% of interactive rates).

### Gemini 2.5 Series

#### Gemini 2.5 Pro
| Tier | Input price (≤200k tokens) | Input price (>200k tokens) | Output price (≤200k tokens) | Output price (>200k tokens) | Context caching (≤200k) | Context caching (>200k) | Storage price | Grounding with Google Search |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Free | Free | - | Free | - | Not available | Not available | - | Not available |
| Paid | $1.25 | $2.50 | $10.00 | $15.00 | $0.31 | $0.625 | $4.50 / 1M tokens per hour | 1,500 RPD free, then $35 / 1k requests |

#### Gemini 2.5 Flash
| Tier | Input price (text/image/video) | Input price (audio) | Output price | Context caching (text/image/video) | Context caching (audio) | Storage price | Grounding | Live API |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Free | Free | Free | Free | Not available | Not available | - | Free up to 500 RPD (shared with Flash-Lite) | Free |
| Paid | $0.30 | $1.00 | $2.50 | $0.075 | $0.25 | $1.00 / 1M tokens per hour | 1,500 RPD free, then $35 / 1k requests | Input: $0.50 (text), $3.00 (audio/image/video); Output: $2.00 (text), $12.00 (audio) |

#### Gemini 2.5 Flash-Lite
| Tier | Input price (text/image/video) | Input price (audio) | Output price | Context caching (text/image/video) | Context caching (audio) | Storage price | Grounding |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Free | Free | Free | Free | Not available | Not available | - | Free up to 500 RPD (shared with Flash) |
| Paid | $0.10 | $0.30 | $0.40 | $0.025 | $0.125 | $1.00 / 1M tokens per hour | 1,500 RPD free, then $35 / 1k requests |

#### Gemini 2.5 Flash Native Audio (Preview)
| Tier | Input price (text) | Input price (audio/video) | Output price (text) | Output price (audio) |
| --- | --- | --- | --- | --- |
| Free | Not available | Not available | Not available | Not available |
| Paid | $0.50 | $3.00 | $2.00 | $12.00 |

#### Gemini 2.5 Flash Image Preview (Preview)
| Tier | Input price (text/image) | Output price |
| --- | --- | --- |
| Free | Not available | Not available |
| Paid | $0.30 | $0.039 per image* |

> *Image output is priced at $30 per 1M tokens. Outputs ≤1024×1024 consume ~1290 tokens (≈$0.039 per image).

#### Gemini 2.5 Flash Preview TTS (Preview)
| Tier | Input price | Output price |
| --- | --- | --- |
| Free | Free | Free |
| Paid | $0.50 (text) | $10.00 (audio) |

#### Gemini 2.5 Pro Preview TTS (Preview)
| Tier | Input price | Output price |
| --- | --- | --- |
| Free | Not available | Not available |
| Paid | $1.00 (text) | $20.00 (audio) |

### Gemini 2.0 Series

#### Gemini 2.0 Flash
| Tier | Input price (text/image/video) | Input price (audio) | Output price | Context caching (text/image/video) | Context caching (audio) | Storage price | Image generation | Tuning | Grounding | Live API |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Free | Free | Free | Free | Free ($0 if ≤ quota) | Free | Not available | Free | Not available | Free up to 500 RPD | Free |
| Paid | $0.10 | $0.70 | $0.40 | $0.025 / 1M tokens | $0.175 / 1M tokens | $1.00 / 1M tokens per hour | $0.039 per image* | Not available | 1,500 RPD free, then $35 / 1k requests | Input: $0.35 (text), $2.10 (audio/image/video); Output: $1.50 (text), $8.50 (audio) |

#### Gemini 2.0 Flash-Lite
| Tier | Input price | Output price | Context caching | Storage | Tuning | Grounding |
| --- | --- | --- | --- | --- | --- | --- |
| Free | Free | Free | Not available | Not available | Not available | Not available |
| Paid | $0.075 | $0.30 | Not available | Not available | Not available | Not available |

### Imagen

#### Imagen 4 (Preview)
| Tier | Fast image price | Standard image price | Ultra image price |
| --- | --- | --- | --- |
| Free | Not available | Not available | Not available |
| Paid | $0.02 | $0.04 | $0.06 |

#### Imagen 3
| Tier | Image price |
| --- | --- |
| Free | Not available |
| Paid | $0.03 |

### Veo (Video Generation)

#### Veo 3
| Tier | Video with audio price (per second) |
| --- | --- |
| Free | Not available |
| Paid | $0.40 |

#### Veo 3 Fast
| Tier | Video with audio price (per second) |
| --- | --- |
| Free | Not available |
| Paid | $0.15 |

> Charges apply only when a video is successfully generated.

#### Veo 2
| Tier | Video price (per second) |
| --- | --- |
| Free | Not available |
| Paid | $0.35 |

### Gemini Embedding
| Tier | Input price |
| --- | --- |
| Free | Free |
| Paid | $0.15 per 1M tokens |

### Gemma

#### Gemma 3
| Tier | Input price | Output price | Context caching | Storage | Tuning | Grounding |
| --- | --- | --- | --- | --- | --- | --- |
| Free | Free | Free | Free | Free | Not available | Not available |
| Paid | Not available | Not available | Not available | Not available | Not available | Not available |

#### Gemma 3n
| Tier | Input price | Output price | Context caching | Storage | Tuning | Grounding |
| --- | --- | --- | --- | --- | --- | --- |
| Free | Free | Free | Free | Free | Not available | Not available |
| Paid | Not available | Not available | Not available | Not available | Not available | Not available |

### Gemini 1.5 Series

#### Gemini 1.5 Flash
| Tier | Input price (≤128k) | Input price (>128k) | Output price (≤128k) | Output price (>128k) | Context caching price (≤128k) | Context caching price (>128k) | Storage price | Tuning |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Free | Free | - | Free | - | Free (≤1M tokens storage per hour) | Free | $1.00 per hour (paid tier) | Token prices same for tuned models; tuning service free |
| Paid | $0.075 | $0.15 | $0.30 | $0.60 | $0.01875 | $0.0375 | $1.00 per hour | Token prices same; tuning service free |

#### Gemini 1.5 Flash-8B
| Tier | Input price (≤128k) | Input price (>128k) | Output price (≤128k) | Output price (>128k) | Context caching (≤128k) | Context caching (>128k) | Storage price | Tuning |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Free | Free | - | Free | - | Free (≤1M tokens storage per hour) | Free | $0.25 per hour (paid tier) | Token prices same; tuning service free |
| Paid | $0.0375 | $0.075 | $0.15 | $0.30 | $0.01 | $0.02 | $0.25 per hour | Token prices same; tuning service free |

#### Gemini 1.5 Pro
| Tier | Input price (≤128k) | Input price (>128k) | Output price (≤128k) | Output price (>128k) | Context caching price (≤128k) | Context caching price (>128k) | Storage price | Tuning |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Free | Free | - | Free | - | Not available | Not available | Not available | Not available |
| Paid | $1.25 | $2.50 | $5.00 | $10.00 | $0.3125 | $0.625 | $4.50 per hour | Not available |

> Grounding with Google Search beyond free allowances costs $35 per 1,000 requests across Gemini paid tiers.

---

## Cognee Integration Notes

### LLM Provider Configuration
- Cognee defaults to OpenAI for LLM operations when only an OpenAI-style API key (e.g., `OPENAI_API_KEY` / `LLM_API_KEY`) is provided.
- Gemini can be used as an LLM provider by setting appropriate Cognee config entries (e.g., `COGNEE_LLM_PROVIDER="gemini"`) and supplying the Gemini API key.
- Cognee supports additional providers (Mistral, Azure OpenAI, Ollama) for generation tasks via pluggable adapters.
- Rate limiting in Cognee’s LLM layer leverages the shared retry handler (`coderisk/coderisk/core/retry_handler.py`) for exponential backoff and provider-specific error handling.

### Embedding Provider Configuration
Cognee allows explicit embedding provider selection through environment variables:
```dotenv
EMBEDDING_PROVIDER="openai"        # or gemini, mistral, ollama, fastembed, custom
EMBEDDING_MODEL="openai/text-embedding-3-large"
EMBEDDING_DIMENSIONS="3072"
EMBEDDING_API_KEY="..."            # falls back to LLM_API_KEY
EMBEDDING_ENDPOINT="https://api.openai.com/v1"
EMBEDDING_MAX_TOKENS="8191"        # optional cap per request
```
- **Gemini embeddings** (`gemini/text-embedding-004`) are available by switching `EMBEDDING_PROVIDER` to `gemini` and specifying the correct dimensions (typically `768`).
- Local and custom providers (Ollama, Fastembed, custom OpenAI-compatible endpoints) can be used for cost or privacy reasons, ensuring `EMBEDDING_DIMENSIONS` matches the vector store schema.
- Cognee’s embedding stack integrates with LanceDB, SQLite, or alternative configured vector stores without additional code changes.

### Rate Limiting & Testing Controls
- Cognee exposes environment toggles for embedding throttling:
  ```dotenv
  EMBEDDING_RATE_LIMIT_ENABLED="true"
  EMBEDDING_RATE_LIMIT_REQUESTS="10"
  EMBEDDING_RATE_LIMIT_INTERVAL="5"
  ```
- `MOCK_EMBEDDING="true"` can be used during testing to return zero vectors and avoid provider charges.
- For LLM calls, Cognee’s retry handler records rate-limit failures and respects provider-directed wait times, enabling budget-aware wrappers in CodeRisk’s risk engines.

### Operational Considerations
- When only one of LLM or embedding providers is configured, Cognee defaults the other to OpenAI—ensure both are set to avoid unexpected cross-billing.
- Cognee’s dataset ingestion and search stack (Kuzu graph + LanceDB vectors + SQLite metadata) can operate entirely locally, aligning with CodeRisk’s financial constraints.
- CodeRisk’s `SearchEnhancedRiskEngine` is poised to incorporate budget governance by querying a centralized `APIBudgetManager` before invoking Cognee searches or OpenAI/Gemini APIs.
- For Gemini usage, remember free tier traffic may be used to improve Google’s models; paid tier disables this, matching enterprise privacy requirements.

---

**Data freshness:** Pricing captured as of the raw listings provided. Providers update rates frequently; confirm against official pricing portals before enforcing budgets in production.
