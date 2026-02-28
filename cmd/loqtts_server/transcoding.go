package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// TranscodeAudio transcodes the wave audio into the specified format using FFmpeg
func TranscodeAudio(audioChan <-chan []byte, outFormat string, ffmpegPath string) (<-chan []byte, error) {
	cmd := exec.Cmd{
		Path:   "ffmpeg",
		Args:   []string{"ffmpeg", "-i", "pipe:0"},
		Stderr: os.Stderr,
	}
	if ffmpegPath != "" {
		cmd.Path = ffmpegPath
	}

	switch outFormat {
	case "wav":
		return nil, fmt.Errorf("programming error: should not call ffmpeg to output wav")
	case "opus", "ogg", "vorbis":
		cmd.Args = append(cmd.Args, "-f", "ogg", "-c:a", "libopus")
	case "mp3":
		cmd.Args = append(cmd.Args, "-f", "mp3")
	case "aac":
		cmd.Args = append(cmd.Args, "-f", "adts", "-c:a", "aac")
	case "flac":
		cmd.Args = append(cmd.Args, "-f", "flac")
	default:
		return nil, fmt.Errorf("unsupported output format: %s", outFormat)
	}
	cmd.Args = append(cmd.Args, "pipe:1")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	log.Debug().Strs("ffmpeg_args", cmd.Args).Msg("starting ffmpeg with arguments")

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		for audio := range audioChan {
			if _, err := stdin.Write(audio); err != nil && !errors.Is(err, io.ErrClosedPipe) {
				log.Panic().Err(err).Msg("error writing to ffmpeg stdin")
			}
		}
		err := stdin.Close()
		if err != nil {
			log.Error().Err(err).Msg("error closing ffmpeg stdin")
		}
		log.Trace().Msg("ffmpeg stdin closed")
	}()

	outChan := make(chan []byte)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				outChan <- chunk
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Panic().Err(err).Msg("error reading from ffmpeg stdout")
			}
		}
		close(outChan)
		log.Trace().Msg("ffmpeg stdout closed")
	}()

	return outChan, nil
}
