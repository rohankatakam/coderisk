# Cognee Integration Setup

## Current Status

CodeRisk is built with **dual-mode support**:
- **Simple Mode** (default): Works immediately with basic file analysis
- **Cognee Mode** (advanced): Full CodeGraph integration with temporal awareness

## Cognee Requirements

Based on https://docs.cognee.ai/getting-started/installation:

### Prerequisites
- **Python 3.9 - 3.12** (Cognee does not support Python 3.13 yet)
- OpenAI API key or other LLM provider
- Virtual environment recommended

### Installation Steps

1. **Create Python 3.10/3.11/3.12 environment**:
   ```bash
   # Using pyenv
   pyenv install 3.11.9
   pyenv virtualenv 3.11.9 coderisk-cognee
   pyenv activate coderisk-cognee

   # Or using uv
   uv venv --python 3.11
   source .venv/bin/activate
   ```

2. **Set up environment variables**:
   ```bash
   # Create .env file in project root
   echo 'LLM_API_KEY="your_openai_api_key"' > .env
   ```

3. **Install CodeRisk with Cognee**:
   ```bash
   # Update requirements.txt to enable Cognee
   # Uncomment the cognee>=0.1.0 line
   pip install cognee>=0.1.0
   pip install -e .
   ```

## Enabling Cognee Mode

Once Cognee is installed properly, CodeRisk will automatically detect it and use the advanced CodeGraph analyzer instead of the simple analyzer.

### Verification

Check if Cognee mode is active:
```bash
crisk check --verbose
```

Look for:
- "Initializing CodeGraph..." (Cognee mode)
- vs "Initializing simple analyzer..." (Simple mode)

## Features Available in Each Mode

### Simple Mode (Current)
✅ **Working now**:
- Basic git diff analysis
- File-based risk scoring
- Regression scaling formula
- CLI interface with rich output
- JSON output for integration

❌ **Limited**:
- No historical pattern analysis
- No dependency graph traversal
- No temporal awareness
- Basic heuristics only

### Cognee Mode (Advanced)
🚀 **Enhanced capabilities**:
- Full CodeGraph dependency analysis
- Historical incident correlation
- Temporal pattern recognition
- Advanced similarity search
- Machine learning-based risk detection
- Continuous learning from feedback

## Migration Path

1. **Week 1 MVP**: Use Simple Mode for immediate value
2. **Week 2+**: Migrate to Cognee Mode for advanced features
3. **Production**: Full Cognee integration with feedback loops

## Development Strategy

For now, continue development with Simple Mode to:
1. Validate core risk assessment logic
2. Test CLI interface and user experience
3. Gather initial feedback
4. Prove product-market fit

Then enhance with Cognee for enterprise features.

## Current Testing Environment

- Python 3.13.5 (incompatible with Cognee)
- Simple analyzer mode active
- All basic functionality working
- Ready for immediate MVP testing

## Next Steps

1. **For MVP testing**: Continue with Simple Mode
2. **For Cognee integration**: Set up Python 3.11 environment
3. **For production**: Full Cognee setup with API keys