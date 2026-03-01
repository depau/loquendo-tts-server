package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// TranscodeAudio transcodes the wave audio into the specified format using FFmpeg
func TranscodeAudio(reader io.ReadCloser, outFormat string, ffmpegPath string) (io.ReadCloser, error) {
	cmd := exec.Command(ffmpegPath, "-i", "pipe:0")
	cmd.Stdin = reader
	cmd.Stderr = os.Stderr
	//if ffmpegPath != "" {
	//	cmd.Path = ffmpegPath
	//}

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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	log.Debug().Strs("ffmpeg_args", cmd.Args).Msg("starting ffmpeg with arguments")

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("ffmpeg exited with error")
		}
		reader.Close()
	}()

	return stdout, nil
}
