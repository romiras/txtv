# txtv

**Precision Truncation for Text Streams: The LLM Guardrail You've Been Waiting For.**

`txtv` is a CLI tool designed for deterministic text stream truncation. It ensures your text fits within strict limits—like LLM context windows—without splitting multi-byte Unicode characters or breaking atomic word boundaries.

## Motivation

Traditional Unix tools like `head` or `wc` operate on bytes or single-byte characters, which makes them unreliable for modern, Unicode-heavy text streams. When feeding data into LLMs, "close enough" isn't good enough. Excess tokens lead to rejected requests, and split Unicode sequences lead to garbled input. `txtv` solves this by using a Unicode-aware "Hybrid Heuristic" for tokenization, ensuring that every truncation is clean and correct.

## Purpose

The primary purpose of `txtv` is to act as a reliable pipe-filter that:

1. **Protects LLM Contexts**: Enforces strict token and line limits.
2. **Preserves Data Integrity**: Never cuts in the middle of a Unicode grapheme cluster or word segment.
3. **Ensures Reliability**: Implements a 1MB safety fail-safe for "soft" stop modes to prevent hanging on rogue streams.
4. **Maintains Performance**: Operates with a constant memory footprint, regardless of input size.

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
