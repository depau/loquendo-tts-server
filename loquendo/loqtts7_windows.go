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
	"strings"
	"unsafe"

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
	fmt.Printf("loading engine DLL from '%s'\n", path)
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
		panic("tts library not initialized")
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
		panic("tts library not initialized")
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

func (t *TTS) GetVoices() ([]string, error) {
	outBuf := make([]byte, 2048)
	err := ttsLib.TTSQuery(t.hSession, ffi_wrapper.TTSQueryObjectVoice, "Id", nil, &outBuf, false, false)
	if err != nil {
		return nil, fmt.Errorf("error querying voices: %v", err)
	}
	// Truncate at the null terminator, if any
	if i := bytes.IndexByte(outBuf, 0); i >= 0 {
		outBuf = outBuf[:i]
	}
	outStr := string(outBuf)
	return strings.Split(outStr, ";"), nil
}

func ttsCallbackWrapper(promptID uint32, eventType ffi_wrapper.TTSEventType, iData uintptr, pUser uintptr) uint32 {
	t := (*TTS)(unsafe.Pointer(pUser))
	t.ttsCallback(promptID, eventType, iData)
	return 0
}

func (t *TTS) ttsCallback(promptID uint32, eventType ffi_wrapper.TTSEventType, iData uintptr) {
	if promptID != t.currentPromptID {
		fmt.Printf("warning: received callback for prompt ID %d, but current prompt ID is %d\n", promptID, t.currentPromptID)
	}
	if t.debugEvents {
		fmt.Printf("tts callback for prompt ID %d: %s\n", promptID, ffi_wrapper.TTSDescribeEvent(eventType, iData))
	}
}

func (t *TTS) SpeakStreaming(text string, voice string) (<-chan []byte, error) {
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

	if err = ttsLib.TTSLoadPersona(t.phReader, voice, nil); err != nil {
		return nil, fmt.Errorf("error loading persona: %v", err)
	}

	ch := make(chan []byte)
	t.speechChannel = ch

	go func() {
		conn, err := t.pipe.Accept()
		if err != nil {
			panic(fmt.Sprintf("error accepting connection on pipe: %v", err))
		}

		defer func() {
			_ = conn.Close()
			_ = t.pipe.Close()
			close(t.speechChannel)
			t.pipe = nil
			t.speechChannel = nil
			fmt.Printf("closed pipe for prompt ID %d\n", t.currentPromptID)
		}()

		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				if errors.Is(err, os.ErrClosed) {
					fmt.Printf("pipe closed, ending read loop for prompt ID %d\n", t.currentPromptID)
					return
				}
				panic(fmt.Sprintf("error reading from pipe: %v", err))
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
