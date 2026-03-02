# Loquendo TTS Server

An unofficial Dockerized wrapper for the Loquendo 7 TTS engine, providing an
OpenAI-compatible REST API for high-quality text-to-speech synthesis.

## Features

- **OpenAI API Compatibility**: Drop-in replacement for OpenAI's `tts-1`
  models (with some exceptions).
- **Architecture Support**: Multi-arch images supporting both `amd64` and
  `arm64`.
- **Streaming Support**: Direct audio streaming for low-latency applications.
- **Multiple Formats**: Supports `mp3`, `opus`, `aac`, `flac`, and `wav`. (`pcm`
  is not supported).
- **Customizable**: Access advanced Loquendo engine parameters via the
  `instructions` field.
- **Web Interface**: Built-in web UI for testing voices and parameters.
- **Dockerized**: Easy deployment with all dependencies (Wine, Ffmpeg) included.

The software is built for Windows 32-bit since that is the only version of the
Loquendo TTS engine available. The x86-64 Docker image packages Wine, and the
ARM64 Docker image uses the [Hangover](https://github.com/AndreRH/hangover)
distribution of Wine to run the Intel code on ARM64.

## Quick Start

The easiest way to run the server is via Docker. The following command runs the
server with the iconic **Roberto** (Trenitalia station) voice.

### Running with Network Admin Capabilities (Recommended)

This is needed for the license check to work by setting up a dummy network
interface.

```bash
docker run --rm -p 8080:8080 --cap-add=NET_ADMIN -it ghcr.io/depau/loquendo-tts-server-roberto:latest
```

### Running with a Specific MAC Address

If you cannot or do not want to use `NET_ADMIN`, you can specify the MAC address
manually:

```bash
docker run --rm -p 8080:8080 --mac-address="7A:8B:9C:1D:2E:3F" -it ghcr.io/depau/loquendo-tts-server-roberto:latest
```

Once running, the web interface is available at `http://localhost:8080/`.

## API Reference

### POST `/v1/audio/speech`

Generates audio from input text. This endpoint is compatible with the OpenAI
API.

**Request Body:**

| Field             | Type   | Description                                                                            |
|:------------------|:-------|:---------------------------------------------------------------------------------------|
| `input`           | string | The text to synthesize.                                                                |
| `model`           | string | The voice model to use (e.g., `tts-loquendo-roberto`).                                 |
| `response_format` | string | Audio format: `mp3` (default), `opus`, `aac`, `flac`, `wav`. (`pcm` is not supported). |
| `speed`           | float  | Speed of the speech (0.25 to 4.0).                                                     |
| `instructions`    | string | Optional line-separated `Key=Value` list of Loquendo parameters.                       |

> [!NOTE]
> The `voice` parameter is currently ignored in favor of `model`.

### GET `/v1/models`

Lists all available voices installed in the container.

## Loquendo Parameters (`instructions`)

You can fine-tune the TTS engine by providing a list of parameters in the
`instructions` field, separated by newlines.

For instance:

```
AutoGuess=VoiceSentence:Italian,English
ProsodicPauses=punctuation
```

| Parameter               | Values / Description                                                                                                 |
|:------------------------|:---------------------------------------------------------------------------------------------------------------------|
| `MultiSpacePause`       | `TRUE` (default), `FALSE` - Whether multiple spaces generate a pause.                                                |
| `MaxParPause`           | Integer - Minimum word count for automatic pause at end of line (default: 5).                                        |
| `ProsodicPauses`        | `automatic`, `punctuation`, `word` - How pauses are inserted.                                                        |
| `ShortPauseLength`      | ms - Duration of short pauses (default: 50).                                                                         |
| `MediumPauseLength`     | ms - Duration of pauses at commas (default: 120).                                                                    |
| `LongPauseLength`       | ms - Duration of end-of-sentence pauses (default: 500).                                                              |
| `SpellingLevel`         | `normal`, `spelling`, `pronounce`.                                                                                   |
| `SpellPunctuation`      | `TRUE`, `FALSE` - Whether to spell out punctuation marks.                                                            |
| `TaggedText`            | `TRUE` (process tags), `FALSE` (pronounce tags) (default `FALSE`). See [Tagged text](#tagged-text) below for details |
| `TextFormat`            | `ssml`, `plain` (default), `autodetect`.                                                                             |
| `DefaultNumberType`     | `generic`, `telephone`, `currency`, `code`, `hour`, `date`, `amount`.                                                |
| `AutoGuess`             | See [Advanced Parameters](#autoguess) below.                                                                         |
| `LanguageSetForGuesser` | Comma-separated list of languages for the Guesser.                                                                   |

### Advanced Parameters

#### Tagged text

The engine supports changing parameters based on tags present in the input text.

The syntax is `\@Param=Value`. Most parameters listed above can be set by
using this syntax.

By default, however, tag processing is disabled. To process tags, set the
`TaggedText` parameter to `TRUE` in the `instructions` field.

#### `AutoGuess`

Activates and configures Mixed Language mode. Syntax:
`AutoGuess=[Type]:[Language list]`.

Possible values for `[Type]` are:

- `no` – no AutoGuess mode
- `VoiceParagraph` – Detects language and changes voice accordingly paragraph by
  paragraph
- `VoiceSentence` - Detects language and changes voice accordingly sentence by
  sentence
- `VoicePhrase` - Detects language and changes voice accordingly phrase by
  phrase
- `LanguageParagraph` – Detects and change language paragraph by paragraph
  without changing the active voice
- `LanguageSentence` – Detects and change language sentence by sentence without
  changing the active voice
- `LanguagePhrase` – Detects and change language phrase by phrase without
  changing the active voice
- `LanguageWord` – Detects and change language word by word without changing the
  active voice
- `BothParagraphSentence` – Combines the effects of `VoiceParagraph` and
  `LanguageSentence`
- `BothParagraphPhrase` – Combines the effects of `VoiceParagraph` and
  `LanguagePhrase`
- `BothParagraphWord` – Combines the effects of `VoiceParagraph` and
  `LanguageWord`
- `BothSentencePhrase` – Combines the effects of `VoiceSentence` and
  `LanguagePhrase`
- `BothSentenceWord` – Combines the effects of `VoiceSentence` and
  `LanguageWord`
- `BothPhraseWord` – Combines the effects of `VoicePhrase` and `LanguageWord`

The AutoGuess keyword requires a comma-separated language list (e.g., English,
French, Spanish, German). The language list may contain also voice names (e.g.,
Dave) as well as language variants (e.g., `Mexican`). For types 9–14 a
postponed `-` (minus) character (e.g. `Swedish-`) means that voice changes are
admitted, but not "language only" changes (see the second example below). A
prefixed `-` (minus) means that only language changes are admitted (not voice
changes). Some examples:

```
AutoGuess=VoiceSentence:Italian,English
// (sentence by sentence changes among Italian and English voices)
AutoGuess=BothSentenceWord:French-,Spanish-,English
// (sentence by sentence detects the right language and changes voice accordingly.
// In addition, while speaking with non-English voices, English words are 
// detected and pronounced with the English phonetic rule set).
```

## Configuration

The server is configured via CLI arguments passed to the entrypoint.

| Argument        | Shortcut | Default  | Description                              |
|:----------------|:---------|:---------|:-----------------------------------------|
| `--addr`        | `-a`     | `:8080`  | Address to listen on.                    |
| `--apikey`      | `-k`     |          | API key for Bearer authentication.       |
| `--log-level`   |          | `info`   | Log level (trace, debug, info, etc.).    |
| `--json-logs`   | `-j`     | `false`  | Output logs in JSON format.              |
| `--debug`       | `-d`     | `false`  | Enable debug logging for the TTS engine. |
| `--ffmpeg-path` |          | `ffmpeg` | Path to the ffmpeg executable.           |

## CLI Usage

A command-line tool `loqtts_speak.exe` is also provided for on-the-fly testing
and generation. It allows listing voices, setting parameters, and outputting
audio to a file.

### Usage

```bash
loqtts_speak.exe --voice Roberto --text "Attenzione! Allontanarsi dalla linea gialla" --output out.wav
```

### Arguments

| Argument              | Shortcut | Default | Description                                        |
|:----------------------|:---------|:--------|:---------------------------------------------------|
| `-t`, `--text`        |          |         | Text to speak (- for stdin).                       |
| `-v`, `--voice`       |          |         | Voice to use.                                      |
| `-s`, `--speed`       |          | `50`    | Speech speed (0-100).                              |
| `-l`, `--list-voices` |          | `false` | List available voices.                             |
| `-p`, `--param`       |          |         | Set engine parameter (can be used multiple times). |
| `-o`, `--output`      |          |         | Output filename (- for stdout).                    |
| `-j`, `--json`        |          | `false` | Output metadata in JSON format.                    |
| `-d`, `--debug`       |          | `false` | Enable debug logging for the TTS engine.           | 

## Development

To build the project locally, you need [Go](https://go.dev/) (version 1.26+) and
`make` installed. Since the Loquendo engine is a 32-bit Windows library, the
binaries must be cross-compiled for `windows/386`. That being said, development
can be done on any platform that supports Go. Just install Wine to try things
out.

```bash
make -j$(nproc)
```

This will produce `loqtts_server.exe` and `loqtts_speak.exe` in the root
directory.

## Credits

- **Web UI**: Forked
  from [openai-tts-client](https://github.com/Cucumber148/openai-tts-client).
- **Wine Bridge**:
  Uses [wine-unix-bridge](https://github.com/depau/wine-unix-bridge) for
  cross-process communication.
- **Base Image**: Built
  on [depau/wine-docker](https://github.com/depau/docker-wine), providing a
  *"lightweight"* Wine environment.

## Disclaimer

This project is an unofficial wrapper and is not affiliated with Loquendo or
Nuance. Using this software requires valid licenses for the Loquendo TTS engine
and voices.
