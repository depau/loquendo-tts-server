package main

import (
	"encoding/json"
	"fmt"
	"io"
	"loq7tts-server/loquendo"
	"loq7tts-server/pkg/utils"
	"os"
	"runtime/debug"

	"github.com/mkideal/cli"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

type argT struct {
	cli.Helper
	Text       string            `cli:"t,text" usage:"Text to speak, - for stdin. Default: the voice's demo sentence" dft:""`
	Voice      string            `cli:"v,voice" usage:"Voice to use"`
	Speed      int32             `cli:"s,speed" usage:"Speech speed in the range 0-100. Default: 50" dft:"50"`
	ListVoices bool              `cli:"l,list-voices" usage:"List available voices" dft:"false"`
	Params     map[string]string `cli:"p,param" usage:"Set a parameter for the voice engine (can be used multiple times), i.e. -pAutoGuess=\"VoiceSentence:Italian,English\"" dft:""`
	JsonOutput bool              `cli:"j,json" usage:"Output JSON instead of plain text (for list-voices)" dft:"false"`
	Output     string            `cli:"o,output" usage:"Output file name, - for stdout" dft:""`
	LogLevel   string            `cli:"log-level" usage:"Log level (trace, debug, info, warn, error, fatal, panic)" dft:"info"`
	DebugTTS   bool              `cli:"d,debug" usage:"enable debug logging for TTS engine events" dft:"false"`
	Version    bool              `cli:"V,version" usage:"show version information" dft:"false"`
}

func main() {
	os.Exit(cli.Run(new(argT), func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)

		if err := utils.SetLogLevel(argv.LogLevel); err != nil {
			return err
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		if argv.Version {
			goLibVersion := "unknown"
			buildInfo, ok := debug.ReadBuildInfo()
			if ok {
				goLibVersion = buildInfo.Main.Version
			}

			loqVersion, err := loquendo.GetVersionInfo()
			if err != nil {
				return err
			}

			if argv.JsonOutput {
				type versionInfo struct {
					GoLibVersion    string `json:"go_lib_version"`
					LoquendoVersion string `json:"loquendo_version"`
				}
				vInfo := versionInfo{
					GoLibVersion:    goLibVersion,
					LoquendoVersion: loqVersion,
				}
				jsonData, err := json.MarshalIndent(vInfo, "", "  ")
				if err != nil {
					return err
				}
				println(string(jsonData))
			} else {
				println("Loquendo TTS wrapper for Go:", goLibVersion)
				println("Loquendo engine version:", loqVersion)
			}

			return nil
		}

		loq, err := loquendo.NewTTS(nil)
		if err != nil {
			return err
		}
		defer loq.Close()

		if argv.DebugTTS {
			loq.SetDebugEvents(true)
		}

		voices, err := loq.GetVoices()
		if err != nil {
			return err
		}

		if argv.ListVoices {
			if argv.JsonOutput {
				jsonData, err := json.MarshalIndent(voices, "", "  ")
				if err != nil {
					return err
				}
				println(string(jsonData))
				return nil
			}

			println("Available voices:")
			for _, v := range voices {
				println(" - Id:", v.Id)
				fmt.Printf("   Native language: %s\n", v.NativeLanguage)
				fmt.Printf("   Gender: %s\n", v.Gender)
				fmt.Printf("   Age: %d\n", v.Age)
				fmt.Printf("   Description: %s\n", v.Description)
			}
			return nil
		}

		if argv.Voice == "" {
			return fmt.Errorf("voice not specified")
		}

		voiceId := argv.Voice
		var voice *loquendo.Voice = nil
		for _, v := range voices {
			if v.Id == voiceId {
				voice = &v
				break
			}
		}
		if voice == nil {
			return fmt.Errorf("voice not found: %s", voiceId)
		}

		text := argv.Text
		if text == "" {
			text = voice.DemoSentence
		} else if text == "-" {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			text = string(bytes)
		}

		for k, v := range argv.Params {
			log.Trace().Str("key", k).Str("value", v).Msg("Setting parameter")
			if err := loq.SetParam(k, v); err != nil {
				return err
			}
		}

		reader, err := loq.SpeakStreaming(text, &loquendo.SpeechOptions{
			Voice: voiceId,
			Speed: &argv.Speed,
		})
		if err != nil {
			return err
		}
		defer reader.Close()

		var output io.Writer
		if argv.Output == "" {
			if term.IsTerminal(int(os.Stdout.Fd())) {
				//goland:noinspection GoErrorStringFormat
				return fmt.Errorf("Binary output can mess up your terminal. Use -o - to write to stdout anyway, or specify an output file with -o <filename>")
			}
			output = os.Stdout
		} else if argv.Output == "-" {
			output = os.Stdout
		} else {
			output, err = os.Create(argv.Output)
			if err != nil {
				return fmt.Errorf("error opening output file: %s", err)
			}
			defer output.(*os.File).Close()
		}
		_, err = io.Copy(output, reader)
		return nil
	}))
}
