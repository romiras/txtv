# txtv

**Streaming Text Truncation: The Lightweight Guardrail for LLM Context Windows.**

`txtv` is a lightning-fast CLI tool designed for deterministic text stream truncation. It ensures your text fits within strict token or line limits without splitting multi-byte Unicode characters or breaking atomic word boundaries—all while maintaining a strictly $\mathcal{O}(1)$ memory footprint.

## Motivation

Traditional Unix tools like `head` or `wc` operate purely on bytes or lines, making them dangerous for modern, Unicode-heavy text streams. When feeding piped data into LLMs, "close enough" isn't good enough. Arbitrarily slicing a stream can lead to rejected requests, garbled input, or split emojis.

Conversely, full-fledged tokenizers (like `tiktoken`) require loading multi-megabyte vocabularies into RAM, destroying the $\mathcal{O}(1)$ streaming nature of standard Unix pipes.

`txtv` solves this by introducing a dependency-free **Hybrid Heuristic** (inspired by UAX #29). It safely segments text at word and grapheme cluster boundaries, acting as a highly efficient, conservative approximation of LLM token limits designed specifically for bash pipelines.

## Key Features

1. **Protects LLM Contexts**: Enforces strict token and line limits natively in bash before you waste API spend.
2. **Preserves Data Integrity**: Safely halts execution without ever cutting a Unicode sequence or word segment in half.
3. **$\mathcal{O}(1)$ Performance**: Streams infinitely large inputs with a constant ~32KB memory footprint. Zero dictionary loading.
4. **Resilient Fail-Safes**: Built-in 1MB safety catch for "soft" stop modes to prevent hanging on unbounded streams.

## Usage

```bash
cat input.txt | txtv [OPTIONS]
```

### Options

- `--max-tokens <N>`: Maximum tokens to emit.
- `--max-lines <N>`: Maximum lines to emit.
- `--soft`: Continue until the current line ends (or 1MB limit hit) when the token limit is reached.
- `--flush`: Enable real-time piping by flushing output after every token.
- `--summary <kv|json|off>`: Control the summary metrics printed to `stderr`.

## Development

The project uses a standard `Makefile` for common tasks.

### Build and Test

- **Build the binary**:
  ```bash
  make build
  ```
- **Run all tests**:
  ```bash
  make test
  ```
- **Run unit tests**:
  ```bash
  make test-unit
  ```
- **Format code**:
  ```bash
  make fmt
  ```
- **Clean artifacts**:
  ```bash
  make clean
  ```

## License

This project is licensed under the [MIT License](LICENSE).
