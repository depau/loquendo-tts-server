package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"loq7tts-server/loquendo"
	"loq7tts-server/pkg/utils"
	"math"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/mkideal/cli"
	"github.com/rs/zerolog"
)

//go:embed web/*
var webContent embed.FS

type argT struct {
	cli.Helper
	BindAddr   string `cli:"a,addr" usage:"address to listen on" dft:":8080"`
	DebugTTS   bool   `cli:"d,debug" usage:"enable debug logging for TTS engine events" dft:"false"`
	ApiKey     string `cli:"k,apikey" usage:"API key for authentication" dft:""`
	FfmpegPath string `cli:"ffmpeg-path" usage:"Path to ffmpeg executable" dft:"ffmpeg"`
	LogLevel   string `cli:"log-level" usage:"Log level (trace, debug, info, warn, error, fatal, panic)" dft:"info"`
	JsonLogs   bool   `cli:"j,json-logs" usage:"Output JSON logs instead of plain text" dft:"false"`
}

func main() {
	os.Exit(cli.Run(new(argT), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		if err := utils.SetLogLevel(argv.LogLevel); err != nil {
			return err
		}
		if argv.JsonLogs {
			log.Logger = log.Output(zerolog.New(os.Stderr).With().Timestamp().Caller().Logger())
		} else {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}
		if err := runServer(argv); err != nil {
			return err
		}
		return nil
	}))
}

func getAvailableVoices() ([]loquendo.Voice, error) {
	loq, err := loquendo.NewTTS(nil)
	if err != nil {
		return nil, err
	}
	defer loq.Close()
	return loq.GetVoices()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func serveSpeech(debugTTS bool, ffmpegPath string, w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Input          string  `json:"input"`
		Model          string  `json:"model"`
		_              any     `json:"voice"`
		Instructions   string  `json:"instructions" default:""`
		ResponseFormat string  `json:"response_format" default:"mp3"`
		Speed          float64 `json:"speed" default:"1.0"`
		StreamFormat   string  `json:"stream_format"`
	}
	var reqBody requestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Err(err).Msg("Error decoding JSON body")
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if !slices.Contains([]string{"mp3", "opus", "aac", "flac", "wav"}, reqBody.ResponseFormat) {
		log.Warn().Str("response_format", reqBody.ResponseFormat).Msg("Unsupported response format")
		http.Error(w, "Unsupported response format", http.StatusBadRequest)
		return
	}

	if reqBody.StreamFormat != "" && reqBody.StreamFormat != "audio" {
		log.Warn().Str("stream_format", reqBody.StreamFormat).Msg("Unsupported stream format")
		http.Error(w, "Unsupported stream format (only 'audio' is supported)", http.StatusNotImplemented)
		return
	}

	if reqBody.Speed < 0 || reqBody.Speed > 4 {
		log.Warn().Float64("speed", reqBody.Speed).Msg("Invalid speed")
		http.Error(w, "Invalid speed (must be between 0 and 4)", http.StatusBadRequest)
		return
	}

	inputVoice := strings.TrimPrefix(reqBody.Model, "tts-loquendo-")
	if inputVoice == "" {
		log.Warn().Str("model", reqBody.Model).Msg("Invalid model")
		http.Error(w, "Invalid model", http.StatusBadRequest)
		return
	}

	loq, err := loquendo.NewTTS(nil)
	if err != nil {
		log.Err(err).Msg("Error initializing TTS engine")
		http.Error(w, "TTS engine error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer loq.Close()

	if debugTTS {
		loq.SetDebugEvents(true)
	}

	voices, err := loq.GetVoices()
	if err != nil {
		log.Err(err).Msg("Error retrieving available voices from TTS engine")
		http.Error(w, "TTS engine error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	voice := ""
	for _, v := range voices {
		if strings.EqualFold(v.Id, inputVoice) {
			voice = v.Id
			break
		}
	}

	if voice == "" {
		log.Warn().Str("voice", inputVoice).Msg("Voice not found")
		http.Error(w, "Requested voice not found: "+inputVoice, http.StatusNotFound)
		return
	}

	var mappedSpeed int32 = 50
	if reqBody.Speed != 1 {
		// Map:  2^x, with x in [-2, +2]
		// To:   [0, 100]
		mappedSpeed = int32(100 * (math.Log2(reqBody.Speed) + 2) / 4)
	}
	log.Debug().Float64("from", reqBody.Speed).Int32("to", mappedSpeed).Msg("Mapped speed")

	for _, line := range strings.Split(reqBody.Instructions, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			log.Warn().Str("line", line).Msg("Invalid instruction line (expected 'key=value')")
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		log.Debug().Str("key", key).Str("value", value).Msg("Setting TTS parameter from instructions")
		if err := loq.SetParam(key, value); err != nil {
			log.Warn().Err(err).Str("key", key).Str("value", value).Msg("Error setting TTS parameter")
			http.Error(w, fmt.Sprintf("Invalid TTS parameter in instructions: '%s'", line), http.StatusBadRequest)
			return
		}
	}

	reader, err := loq.SpeakStreaming(reqBody.Input, &loquendo.SpeechOptions{
		Voice: voice,
		Speed: &mappedSpeed,
	})
	if err != nil {
		log.Err(err).Msg("Error starting TTS streaming")
		http.Error(w, "TTS engine error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	mimeType := fmt.Sprintf("audio/%s", reqBody.ResponseFormat)
	fileExt := reqBody.ResponseFormat
	if reqBody.ResponseFormat == "ogg" {
		mimeType = "audio/ogg"
		fileExt = "oga"
	}
	if reqBody.ResponseFormat != "wav" {
		newReader, err := TranscodeAudio(reader, reqBody.ResponseFormat, ffmpegPath)
		if err != nil {
			log.Error().Err(err).Msg("Error transcoding audio")
			http.Error(w, "Error transcoding audio", http.StatusInternalServerError)
			reader.Close()
			return
		}
		reader = newReader
	}
	defer reader.Close()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=tts.%s", fileExt))
	w.WriteHeader(http.StatusOK)

	if _, err = io.Copy(w, reader); err != nil {
		log.Error().Err(err).Msg("Error writing audio to response")
		http.Error(w, "Error writing audio to response", http.StatusInternalServerError)
		return
	}
}

func runServer(argv *argT) error {
	// Prepare the list of available voices
	voices, err := getAvailableVoices()
	if err != nil {
		return err
	}
	log.Debug().Str("voices", fmt.Sprintf("%+v", voices)).Msg("Available voices")

	models := make(map[string]any)
	data := make([]map[string]string, 0)

	models["object"] = "list"
	for _, v := range voices {
		data = append(data, map[string]string{
			"id":              fmt.Sprintf("tts-loquendo-%s", strings.ToLower(v.Id)),
			"name":            v.Id,
			"native_language": v.NativeLanguage,
			"gender":          v.Gender,
			"object":          "model",
		})
	}
	models["data"] = data

	apiKeyMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if argv.ApiKey != "" {
				key := r.Header.Get("Authorization")
				if key != "Bearer "+argv.ApiKey {
					log.Warn().Str("key", key).Msg("Invalid API key")
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, models)
	})

	mux.Handle("POST /v1/audio/speech", apiKeyMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		serveSpeech(argv.DebugTTS, argv.FfmpegPath, writer, request)
	})))

	webFS := mustSub(webContent, "web")
	mux.Handle("GET /web/", cacheStatic(http.StripPrefix("/web/", http.FileServer(http.FS(webFS)))))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/web/", http.StatusFound)
	})

	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	logRequestsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().Str("method", r.Method).Str("url", r.URL.Path).Msg("Request received")
			next.ServeHTTP(w, r)
		})
	}

	log.Info().Str("addr", argv.BindAddr).Msg("Starting server")
	if err := http.ListenAndServe(argv.BindAddr, logRequestsMiddleware(corsMiddleware(mux))); err != nil {
		return err
	}

	return nil
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		log.Panic().Err(err).Str("dir", dir).Msg("Error creating sub filesystem")
	}
	return sub
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Long cache for embedded versioned assets; for quick iteration, set to no-store.
		w.Header().Set("Cache-Control", "public, max-age=86400")
		next.ServeHTTP(w, r)
	})
}
