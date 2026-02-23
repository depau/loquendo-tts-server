package main

import (
	"loq7tts-server/loquendo"
	"os"
)

func main() {
	ver, err := loquendo.GetVersionInfo()
	if err != nil {
		println("error")
		panic(err)
	}
	println("Loquendo LTTS7 version:", ver)

	loq, err := loquendo.NewTTS(nil)
	if err != nil {
		panic(err)
	}
	defer loq.Close()

	voices, err := loq.GetVoices()
	if err != nil {
		panic(err)
	}
	for _, v := range voices {
		println(v)
	}

	dataChan, err := loq.SpeakStreaming("Ciao, questo è un test del motore Loquendo LTTS7.", "Roberto")
	if err != nil {
		panic(err)
	}

	// Read the channel fully, writing the audio data to a file
	var audioData []byte
	for chunk := range dataChan {
		audioData = append(audioData, chunk...)
	}

	if err := os.WriteFile("output.wav", audioData, 0644); err != nil {
		panic(err)
	}
}
