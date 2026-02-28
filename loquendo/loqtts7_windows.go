//go:build windows

package loquendo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"loq7tts-server/loquendo/ffi_wrapper"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"

	"github.com/natefinch/npipe"
)

var ttsLib *ffi_wrapper.TTSLibrary = nil

type TTS struct {
	phReader ffi_wrapper.TTSHandle
	hSession ffi_wrapper.TTSHandle

	currentPromptID uint32
	speechChannel   chan<- []byte
	pipe            *npipe.PipeListener

	channels   ffi_wrapper.TTSAudioSampleType
	sampleRate uint32

	debugEvents bool
}

type Voice struct {
	Id             string `json:"id"`              // Id is the unique voice identifier
	Description    string `json:"description"`     // Description is the voice mnemonic description
	Gender         string `json:"gender"`          // Gender is the voice gender
	Age            int    `json:"age"`             // Age is the voice age
	NativeLanguage string `json:"native_language"` // NativeLanguage is the voice's native language
	DemoSentence   string `json:"demo_sentence"`   // DemoSentence is a sample sentence in which the voice introduces itself using its native language
	BaseSpeed      int    `json:"base_speed"`      // BaseSpeed is the voice default speech in word/minute
	BasePitch      int    `json:"base_pitch"`      // BasePitch is the voice default pitch in hertz
}

func InitEngineDLL(dllPath *string) (err error) {
	if ttsLib != nil {
		return errors.New("library DLL already initialized")
	}
	var path string
	if dllPath != nil && *dllPath != "" {
		path = *dllPath
	} else {
		path, err = ffi_wrapper.GetDefaultEngineLibPath()
		if err != nil {
			return fmt.Errorf("error finding engine DLL path: %v", err)
		}
	}
	log.Debug().Str("path", path).Msg("loading engine DLL")
	ttsLib, err = ffi_wrapper.LoadEngineDLL(path)
	return err
}

func GetVersionInfo() (string, error) {
	if ttsLib == nil {
		if err := InitEngineDLL(nil); err != nil {
			return "", err
		}
	}
	if ttsLib == nil {
		log.Panic().Msg("tts library not initialized")
	}
	return ttsLib.TTSGetVersionInfo()
}

func NewTTS(iniFile *string) (*TTS, error) {
	if ttsLib == nil {
		if err := InitEngineDLL(nil); err != nil {
			return nil, fmt.Errorf("error initializing TTS library: %v", err)
		}
	}
	if ttsLib == nil {
		log.Panic().Msg("tts library not initialized")
	}

	session, err := ttsLib.TTSNewSession(iniFile)
	if err != nil {
		return nil, fmt.Errorf("error creating TTS session: %v", err)
	}

	reader, err := ttsLib.TTSNewReader(session)
	if err != nil {
		_ = ttsLib.TTSDeleteSession(session)
		return nil, fmt.Errorf("error creating TTS reader: %v", err)
	}

	res := &TTS{
		phReader:        reader,
		hSession:        session,
		currentPromptID: 0,
		speechChannel:   nil,
		channels:        2,
		sampleRate:      32000,
	}

	if err = ttsLib.TTSEnableEvent(reader, ffi_wrapper.TTSEventData, true); err != nil {
		_ = res.Close()
		return nil, fmt.Errorf("error enabling TTS data events: %v", err)
	}

	if err = ttsLib.TTSEnableEvent(reader, ffi_wrapper.TTSEventFreeSpace, false); err != nil {
		_ = res.Close()
		return nil, fmt.Errorf("error disabling TTS free space events: %v", err)
	}

	if err = ttsLib.TTSSetTextEncodingUTF8(res.phReader); err != nil {
		_ = res.Close()
		return nil, fmt.Errorf("error setting text encoding: %v", err)
	}

	if err = ttsLib.TTSSetCallback(reader, ttsCallbackWrapper, uintptr(unsafe.Pointer(res))); err != nil {
		_ = res.Close()
		return nil, fmt.Errorf("error setting TTS callback: %v", err)
	}

	return res, nil
}

func (t *TTS) Close() error {
	if t.phReader != 0 {
		if err := ttsLib.TTSDeleteReader(t.phReader); err != nil {
			return fmt.Errorf("error tearing down TTS reader: %v", err)
		}
		t.phReader = 0
	}
	if t.hSession != 0 {
		if err := ttsLib.TTSDeleteSession(t.hSession); err != nil {
			return fmt.Errorf("error tearing down TTS session: %v", err)
		}
		t.hSession = 0
	}
	return nil
}

func (t *TTS) SetDebugEvents(enabled bool) {
	t.debugEvents = enabled
}

func (t *TTS) SetAudioSettings(sampleRate uint, mono bool) {
	if mono {
		t.channels = ffi_wrapper.TTSAudioSampleTypeMono
	} else {
		t.channels = ffi_wrapper.TTSAudioSampleTypeStereo
	}
	t.sampleRate = uint32(sampleRate)
}

func fixStringEncoding(input string) (string, error) {
	inputBytes := []byte(input)
	e, _, _ := charset.DetermineEncoding(inputBytes, "")
	reader := transform.NewReader(bytes.NewReader(inputBytes), e.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (t *TTS) GetVoices() ([]Voice, error) {
	outBuf := make([]byte, 2048)
	err := ttsLib.TTSQuery(t.hSession, ffi_wrapper.TTSQueryObjectVoice, "Id,Description,Gender,Age,MotherTongue,BaseSpeed,BasePitch,DemoSentence", nil, &outBuf, false, false)
	if err != nil {
		return nil, fmt.Errorf("error querying voices: %v", err)
	}
	// Truncate at the null terminator, if any
	if i := bytes.IndexByte(outBuf, 0); i >= 0 {
		outBuf = outBuf[:i]
	}
	outStr := string(outBuf)
	lines := strings.Split(outStr, ";")

	voices := make([]Voice, len(lines))
	for i, line := range lines {
		parts := strings.Split(line, ",")

		id := parts[0]
		description := parts[1]
		gender := parts[2]
		age, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, fmt.Errorf("error parsing voice age: %v", err)
		}
		motherTongue := parts[4]
		baseSpeed, err := strconv.Atoi(parts[5])
		if err != nil {
			return nil, fmt.Errorf("error parsing voice base speed: %v", err)
		}
		basePitch, err := strconv.Atoi(parts[6])
		if err != nil {
			return nil, fmt.Errorf("error parsing voice base pitch: %v", err)
		}
		demoSentence, err := fixStringEncoding(strings.Join(parts[7:], ","))
		if err != nil {
			return nil, fmt.Errorf("error parsing voice demo sentence: %v", err)
		}

		voices[i] = Voice{
			Id:             id,
			Description:    description,
			Gender:         gender,
			Age:            age,
			NativeLanguage: motherTongue,
			BaseSpeed:      baseSpeed,
			BasePitch:      basePitch,
			DemoSentence:   demoSentence,
		}
	}
	return voices, nil
}

func (t *TTS) SetParam(name, value string) error {
	err := ttsLib.TTSSetParam(t.phReader, name, value)
	if err != nil {
		return fmt.Errorf("error setting TTS parameter: %v", err)
	}
	return nil
}

func ttsCallbackWrapper(promptID uint32, eventType ffi_wrapper.TTSEventType, iData uintptr, pUser uintptr) uint32 {
	//goland:noinspection GoVetUnsafePointer
	t := (*TTS)(unsafe.Pointer(pUser))
	t.ttsCallback(promptID, eventType, iData)
	return 0
}

func (t *TTS) ttsCallback(promptID uint32, eventType ffi_wrapper.TTSEventType, iData uintptr) {
	if promptID != t.currentPromptID {
		log.Warn().Uint32("promptID", promptID).Uint32("currentPromptID", t.currentPromptID).Msg("received callback for prompt ID that does not match current prompt ID")
	}
	if t.debugEvents {
		log.Debug().Uint32("promptID", promptID).Str("event", ffi_wrapper.TTSDescribeEvent(eventType, iData)).Msg("tts callback")
	}
}

type SpeechOptions struct {
	Voice string `json:"voice"`
	Speed *int32 `json:"speed"`
}

func (t *TTS) SpeakStreaming(text string, options *SpeechOptions) (<-chan []byte, error) {
	var err error

	if t.currentPromptID != 0 {
		return nil, errors.New("already speaking")
	}

	// The .wav extension is important, otherwise the engine will write raw PCM data without a header
	randInt := rand.Int()
	pipeName := fmt.Sprintf(`\\.\pipe\loq7tts_pipe_%d_%d_%d.wav`, os.Getpid(), t.currentPromptID, randInt)
	if t.pipe, err = npipe.Listen(pipeName); err != nil {
		return nil, fmt.Errorf("error creating WAV data named pipe: %v", err)
	}

	if err = ttsLib.TTSSetAudio(t.phReader, new("LTTS7AudioFile"), &pipeName, t.sampleRate, ffi_wrapper.TTSAudioEncTypeLinear, t.channels, 0); err != nil {
		return nil, fmt.Errorf("error setting audio output: %v", err)
	}

	if options != nil {
		if err = ttsLib.TTSLoadPersona(t.phReader, options.Voice, nil); err != nil {
			return nil, fmt.Errorf("error loading persona: %v", err)
		}
		var speed int32 = 50
		if options.Speed != nil {
			speed = *options.Speed
		}
		if err = ttsLib.TTSSetSpeed(t.phReader, speed); err != nil {
			return nil, fmt.Errorf("error setting speed: %v", err)
		}
	}

	ch := make(chan []byte)
	t.speechChannel = ch

	go func() {
		conn, err := t.pipe.Accept()
		if err != nil {
			log.Panic().Err(err).Msg("error accepting connection on pipe")
		}

		defer func() {
			_ = conn.Close()
			_ = t.pipe.Close()
			close(t.speechChannel)
			t.pipe = nil
			t.speechChannel = nil
			log.Trace().Str("pipe", pipeName).Msg("pipe closed")
		}()

		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				if errors.Is(err, os.ErrClosed) {
					log.Debug().Str("pipe", pipeName).Msg("pipe closed, ending read loop")
					return
				}
				log.Panic().Err(err).Msg("error reading from pipe")
			}
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				t.speechChannel <- chunk
			}
		}
	}()

	promptId, err := ttsLib.TTSRead(t.phReader, text, true, false)
	if err != nil {
		return nil, fmt.Errorf("error starting TTS read: %v", err)
	}

	t.currentPromptID = promptId

	return ch, nil
}
