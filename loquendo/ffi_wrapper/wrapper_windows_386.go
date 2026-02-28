//go:build windows && 386

package ffi_wrapper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func (l *TTSLibrary) TTSGetErrorMessage(code TTSResult) string {
	result, _, _ := l.executor.CallProc(l.ttsGetErrorMessage, uintptr(code))
	if result == 0 {
		return fmt.Sprintf("tts error %d", code)
	}
	return windows.BytePtrToString((*byte)(unsafe.Pointer(result)))
}

func (l *TTSLibrary) wrapErr(rc TTSResult) error {
	if rc == TTSOk {
		return nil
	}
	return fmt.Errorf("tts library error: %s", l.TTSGetErrorMessage(rc))
}

func (l *TTSLibrary) TTSNewSession(iniFile *string) (TTSHandle, error) {
	var handle TTSHandle

	if iniFile == nil {
		rc, _, _ := l.executor.CallProc(l.ttsNewSession,
			uintptr(unsafe.Pointer(&handle)),
			0,
		)
		return handle, l.wrapErr(TTSResult(rc))
	}

	iniFilePtr, err := windows.BytePtrFromString(*iniFile)
	if err != nil {
		return 0, err
	}
	rc, _, _ := l.executor.CallProc(l.ttsNewSession,
		uintptr(unsafe.Pointer(&handle)),
		uintptr(unsafe.Pointer(iniFilePtr)),
	)
	return handle, l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSDeleteSession(session TTSHandle) error {
	rc, _, _ := l.executor.CallProc(l.ttsDeleteSession, uintptr(session))
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSNewReader(session TTSHandle) (TTSHandle, error) {
	var handle TTSHandle
	rc, _, _ := l.executor.CallProc(l.ttsNewReader,
		uintptr(unsafe.Pointer(&handle)),
		uintptr(session),
	)
	return handle, l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSDeleteReader(reader TTSHandle) error {
	rc, _, _ := l.executor.CallProc(l.ttsDeleteReader, uintptr(reader))
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSLoadPersona(reader TTSHandle, voice string, language *string) error {
	var (
		voicePtr    uintptr = 0
		languagePtr uintptr = 0
	)
	vp, err := windows.BytePtrFromString(voice)
	if err != nil {
		return err
	}
	voicePtr = uintptr(unsafe.Pointer(vp))
	if language != nil {
		lp, err := windows.BytePtrFromString(*language)
		if err != nil {
			return err
		}
		languagePtr = uintptr(unsafe.Pointer(lp))
	}
	rc, _, _ := l.executor.CallProc(l.ttsLoadPersona,
		uintptr(reader),
		voicePtr,
		languagePtr,
		0, // style parameter is reserved and should be NULL according to the documentation
	)
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSSetAudio(reader TTSHandle, destName *string, deviceName *string, sampleRate uint32, encoding TTSAudioEncodingType, sampleType TTSAudioSampleType, userptr uintptr) error {
	var (
		destNamePtr   uintptr = 0
		deviceNamePtr uintptr = 0
	)
	if destName != nil {
		dnp, err := windows.BytePtrFromString(*destName)
		if err != nil {
			return err
		}
		destNamePtr = uintptr(unsafe.Pointer(dnp))
	}
	if deviceName != nil {
		dnp, err := windows.BytePtrFromString(*deviceName)
		if err != nil {
			return err
		}
		deviceNamePtr = uintptr(unsafe.Pointer(dnp))
	}
	rc, _, _ := l.executor.CallProc(l.ttsSetAudio,
		uintptr(reader),
		destNamePtr,
		deviceNamePtr,
		uintptr(sampleRate),
		uintptr(encoding),
		uintptr(sampleType),
		userptr,
	)
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSRead(reader TTSHandle, input string, async bool, fromFile bool) (promptId uint32, err error) {
	inputPtr, err := windows.BytePtrFromString(input)
	if err != nil {
		return 0, err
	}
	var promptIdOut uint32
	rc, _, _ := l.executor.CallProc(l.ttsRead,
		uintptr(reader),
		uintptr(unsafe.Pointer(inputPtr)),
		uintptr(ToTTSBool(async)),
		uintptr(ToTTSBool(fromFile)),
		uintptr(unsafe.Pointer(&promptIdOut)),
	)
	return promptIdOut, l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSSetCallback(reader TTSHandle, callback TTSCallbackFunctionType, userptr uintptr) error {
	callbackPtr := windows.NewCallbackCDecl(callback)
	rc, _, _ := l.executor.CallProc(l.ttsSetCallback,
		uintptr(reader),
		callbackPtr,
		userptr,
		0, // = TTSCALLBACKFUNCTION
	)
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSSetSpeed(reader TTSHandle, speed int32) error {
	rc, _, _ := l.executor.CallProc(l.ttsSetSpeed,
		uintptr(reader),
		uintptr(speed),
	)
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSQuery(session TTSHandle, queryType TTSQueryType, dataToRetrieve string, filter *string, resultBuffer *[]byte, loadedOnly bool, rescanFileSystem bool) error {
	if resultBuffer == nil || len(*resultBuffer) == 0 {
		return errors.New("resultBuffer must be a non-empty byte slice")
	}
	dataToRetrievePtr, err := windows.BytePtrFromString(dataToRetrieve)
	if err != nil {
		return err
	}
	var filterPtr uintptr = 0
	if filter != nil {
		fp, err := windows.BytePtrFromString(*filter)
		if err != nil {
			return err
		}
		filterPtr = uintptr(unsafe.Pointer(fp))
	}
	resultBufferPtr := uintptr(unsafe.Pointer(&(*resultBuffer)[0]))
	rc, _, _ := l.executor.CallProc(l.ttsQuery,
		uintptr(session),
		uintptr(queryType),
		uintptr(unsafe.Pointer(dataToRetrievePtr)),
		filterPtr,
		resultBufferPtr,
		uintptr(len(*resultBuffer)),
		uintptr(ToTTSBool(loadedOnly)),
		uintptr(ToTTSBool(rescanFileSystem)),
	)
	if err := l.wrapErr(TTSResult(rc)); err != nil {
		return err
	}
	return nil
}

func (l *TTSLibrary) TTSSetParam(readerOrSession TTSHandle, name, value string) error {
	namePtr, err := windows.BytePtrFromString(name)
	if err != nil {
		return err
	}
	valuePtr, err := windows.BytePtrFromString(value)
	if err != nil {
		return err
	}
	rc, _, _ := l.executor.CallProc(l.ttsSetParam,
		uintptr(readerOrSession),
		uintptr(unsafe.Pointer(namePtr)),
		uintptr(unsafe.Pointer(valuePtr)),
	)
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSGetPCM(object TTSHandle) (buffer []byte, complete bool, err error) {
	var bufferPtr uintptr
	var numSamplesOut uint32
	var completeOut TTSBool
	rc, _, _ := l.executor.CallProc(l.ttsGetPCM,
		uintptr(object),
		uintptr(unsafe.Pointer(&bufferPtr)),
		uintptr(unsafe.Pointer(&numSamplesOut)),
		uintptr(unsafe.Pointer(&completeOut)),
	)
	if err := l.wrapErr(TTSResult(rc)); err != nil {
		return nil, false, err
	}
	buffer = unsafe.Slice((*byte)(unsafe.Pointer(bufferPtr)), numSamplesOut)
	return buffer, completeOut == TTSTrue, nil
}

func (l *TTSLibrary) TTSGetVersionInfo() (string, error) {
	var buf [512]byte
	rc, _, _ := l.executor.CallProc(l.ttsGetVersionInfo, uintptr(unsafe.Pointer(&buf[0])))
	if err := l.wrapErr(TTSResult(rc)); err != nil {
		return "", err
	}
	return windows.BytePtrToString(&buf[0]), nil
}

func (l *TTSLibrary) TTSSetTextEncodingUTF8(reader TTSHandle) error {
	// Let's use UTF-8 everywhere since it's the most common encoding and Go's native string encoding.
	rc, _, _ := l.executor.CallProc(l.ttsSetTextEncoding,
		uintptr(reader),
		uintptr(65001), // TTSUTF8 = 65001
	)
	return l.wrapErr(TTSResult(rc))
}

func (l *TTSLibrary) TTSEnableEvent(reader TTSHandle, eventType TTSEventType, enabled bool) error {
	rc, _, _ := l.executor.CallProc(l.ttsEnableEvent,
		uintptr(reader),
		uintptr(eventType),
		uintptr(ToTTSBool(enabled)),
	)
	return l.wrapErr(TTSResult(rc))
}

func findEngineLibPathFromRegistry() (string, error) {
	for _, subkey := range []string{
		`SOFTWARE\Loquendo\LTTS7\Engine`,
		`SOFTWARE\Loquendo\LTTS7\SDK`,
		`SOFTWARE\Loquendo\LTTS7\LoqSAPI5`,
		`SOFTWARE\Loquendo\LTTS7\default.session`,
	} {
		for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
			k, err := registry.OpenKey(root, subkey, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			defer k.Close()

			s, _, err := k.GetStringValue("DataPath")
			if err != nil {
				continue
			}
			s = strings.TrimSpace(strings.Trim(s, `\`))
			if strings.HasSuffix(s, `\Data`) {
				s = strings.TrimSuffix(s, `\Data`)
			}
			if s != "" {
				// Often ends with a trailing slash already; keep as-is for filepath.Join behavior.
				return s, nil
			}
		}
	}
	fallback := `C:\Program Files (x86)\Loquendo\LTTS7\bin\LoqTTS7.dll`
	if _, err := os.Stat(fallback); err == nil {
		return filepath.Dir(fallback), nil
	}
	return "", errors.New("engine path not found")
}

func GetDefaultEngineLibPath() (string, error) {
	dataPath, err := findEngineLibPathFromRegistry()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataPath, "bin", "LoqTTS7.dll"), nil
}
