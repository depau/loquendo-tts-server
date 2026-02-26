package main

import (
	"encoding/json"
	"fmt"
	"loq7tts-server/loquendo"
	"net/http"
	"os"
	"strings"

	"github.com/mkideal/cli"
)

type argT struct {
	cli.Helper
	BindAddr string `cli:"a,addr" usage:"address to listen on" dft:":8080"`
	DebugTTS bool   `cli:"d,debug" usage:"enable debug logging for TTS engine events" dft:"false"`
	ApiKey   string `cli:"k,apikey" usage:"API key for authentication" dft:""`
}

func main() {
	os.Exit(cli.Run(new(argT), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		if err := runServer(argv); err != nil {
			panic(err)
		}
		return nil
	}))
}

func getAvailableVoices() ([]string, error) {
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

func serveSpeech(debugTTS bool, w http.ResponseWriter, r *http.Request) {
	type requestBody struct {
		Input          string  `json:"input"`
		Model          string  `json:"model"`
		_              any     `json:"voice"`
		_              string  `json:"instructions"`
		ResponseFormat string  `json:"response_format"`
		Speed          float64 `json:"speed"`
		StreamFormat   string  `json:"stream_format"`
	}
	var reqBody requestBody
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		println(err.Error())
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if reqBody.ResponseFormat != "wav" {
		http.Error(w, "Unsupported response format (only 'wav' is supported)", http.StatusBadRequest)
		return
	}

	if reqBody.StreamFormat != "" && reqBody.StreamFormat != "audio" {
		http.Error(w, "Unsupported stream format (only 'audio' is supported)", http.StatusBadRequest)
		return
	}

	inputVoice := strings.TrimPrefix(reqBody.Model, "tts-loquendo-")
	if inputVoice == "" {
		http.Error(w, "Invalid model", http.StatusBadRequest)
		return
	}

	loq, err := loquendo.NewTTS(nil)
	if err != nil {
		println(err.Error())
		http.Error(w, "TTS engine error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer loq.Close()

	if debugTTS {
		loq.SetDebugEvents(true)
	}

	voices, err := loq.GetVoices()
	if err != nil {
		println(err.Error())
		http.Error(w, "TTS engine error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	voice := ""
	for _, v := range voices {
		if strings.EqualFold(v, inputVoice) {
			voice = v
			break
		}
	}

	if voice == "" {
		http.Error(w, "Requested voice not found: "+inputVoice, http.StatusNotFound)
		return
	}

	dataChan, err := loq.SpeakStreaming(reqBody.Input, voice)
	if err != nil {
		println(err.Error())
		http.Error(w, "TTS engine error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/wav")
	w.WriteHeader(http.StatusOK)
	for chunk := range dataChan {
		if _, err := w.Write(chunk); err != nil {
			println(err.Error())
			fmt.Printf("Error writing audio chunk to response: %v\n", err)
			return
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func runServer(argv *argT) error {
	// Prepare the list of available voices
	voices, err := getAvailableVoices()
	if err != nil {
		return err
	}
	println("Available voices:")
	for _, v := range voices {
		println(" -", v)
	}

	models := make(map[string]any)
	data := make([]map[string]string, 0)

	models["object"] = "list"
	for _, v := range voices {
		data = append(data, map[string]string{
			"id":     fmt.Sprintf("tts-loquendo-%s", strings.ToLower(v)),
			"object": "model",
		})
	}
	models["data"] = data

	apiKeyMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if argv.ApiKey != "" {
				key := r.Header.Get("Authorization")
				if key != "Bearer "+argv.ApiKey {
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
		serveSpeech(argv.DebugTTS, writer, request)
	})))

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
			fmt.Printf("%s %s\n", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}

	fmt.Printf("Starting server on %s...\n", argv.BindAddr)
	if err := http.ListenAndServe(argv.BindAddr, logRequestsMiddleware(corsMiddleware(mux))); err != nil {
		return err
	}

	return nil
}
