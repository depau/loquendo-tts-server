//go:build windows && 386

package ffi_wrapper

import (
	"fmt"
	"loq7tts-server/loquendo/ffi_wrapper/threads"
	"path/filepath"

	"golang.org/x/sys/windows"
)

type TTSLibrary struct {
	executor *threads.ThreadExecutor
	dll      *windows.DLL

	/*
	   #define ttsSTRINGMAXLEN 512
	   #define tts_API_DEFINITION __declspec(dllexport) _stdcall
	   #define tts_API_DEFINITION_P __declspec(dllexport) * __stdcall
	*/

	/*
		ttsResultType tts_API_DEFINITION ttsNewSession(
		    ttsHandleType *phSession,
		    const char *IniFile
		);
	*/
	ttsNewSession *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsDeleteSession(
		    ttsHandleType hSession
		);
	*/
	ttsDeleteSession *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsNewReader(
		    ttsHandleType *phReader,
		    ttsHandleType hSession
		);
	*/
	ttsNewReader *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsDeleteReader(
		    ttsHandleType hReader
		);
	*/
	ttsDeleteReader *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsLoadPersona(
		    ttsHandleType hReader,
		    const char *szVoice,
		    const char *szLanguage,
		    const char *szStyle
		);
	*/
	ttsLoadPersona *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsSetAudio(
		    ttsHandleType hReader,
		    const char *AudioDestName,
		    const char *AudioDeviceName,
		    unsigned int SampleRate,
		    ttsAudioEncodingType coding,
		    ttsAudioSampleType nChannels,
		    const void *pUser
		);
	*/
	ttsSetAudio *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsRead
		(
			ttsHandleType hReader,
			const void *Input,
			ttsBoolType bAsync,
			ttsBoolType bFromFile,
			unsigned long *phPromptId  // out
		);
	*/
	ttsRead *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsSetCallback(
		    ttsHandleType hReader,
		    void *pCallback,
		    void *pUser,
		    ttsCallbackType Type
		);
	*/
	ttsSetCallback *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsSetSpeed(
		    ttsHandleType hReader,
		    int value
		);
	*/
	ttsSetSpeed *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsQuery(
		    ttsHandleType hSession,
		    ttsQueryType uObject,
		    const char *sDataToRetrieve,
		    const char *sFilter,  // may be NULL
		    char *sResultBuffer,  // out
		    unsigned int uResultBufferSize,
		    ttsBoolType bLoadedOnly,
		    ttsBoolType bRescanFileSystem
		);
	*/
	ttsQuery *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsGetPCM(
		    ttsHandleType hObject,
		    const void **pBuffer,  // out
		    unsigned int *pnSamples,  // out
		    ttsBoolType *bComplete  // out
		);
	*/
	ttsGetPCM *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsSetTextEncoding(
		    ttsHandleType hReader,
		    ttsTextEncodingType encoding
		);
	*/
	ttsSetTextEncoding *windows.Proc

	/*
		ttsResultType tts_API_DEFINITION ttsEnableEvent(
		    ttsHandleType hReader,
		    ttsEventType evt,
		    ttsBoolType bEnabled
		);
	*/
	ttsEnableEvent *windows.Proc

	/*
		const char tts_API_DEFINITION_P ttsGetErrorMessage(
		    ttsResultType ErrNo
		);
	*/
	ttsGetErrorMessage *windows.Proc

	/*
		typedef char ttsInfoStringType[ttsSTRINGMAXLEN];
		ttsResultType tts_API_DEFINITION ttsGetVersionInfo(
			ttsInfoStringType StrVer
		);
	*/
	ttsGetVersionInfo *windows.Proc
}

func LoadEngineDLL(dllPath string) (*TTSLibrary, error) {
	absPath, err := filepath.Abs(dllPath)
	if err != nil {
		return nil, fmt.Errorf("abs dll path: %w", err)
	}

	// Prefer safe search behavior; additionally search the DLL's own directory
	// for its dependencies (e.g. LTTS7Util.dll).
	h, err := windows.LoadLibraryEx(
		absPath,
		0,
		windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_DEFAULT_DIRS,
	)
	if err != nil {
		return nil, fmt.Errorf("LoadLibraryEx(%q): %w", absPath, err)
	}

	dll := &windows.DLL{
		Name:   filepath.Base(absPath),
		Handle: h,
	}

	mustProc := func(name string) (*windows.Proc, error) {
		proc, e := dll.FindProc(name)
		if e != nil {
			_ = windows.FreeLibrary(h)
			return nil, fmt.Errorf("FindProc(%q): %w", name, e)
		}
		return proc, nil
	}

	lib := &TTSLibrary{
		dll:      dll,
		executor: threads.NewExecutor(64),
	}

	if lib.ttsNewSession, err = mustProc("ttsNewSession"); err != nil {
		return nil, err
	}
	if lib.ttsDeleteSession, err = mustProc("ttsDeleteSession"); err != nil {
		return nil, err
	}
	if lib.ttsNewReader, err = mustProc("ttsNewReader"); err != nil {
		return nil, err
	}
	if lib.ttsDeleteReader, err = mustProc("ttsDeleteReader"); err != nil {
		return nil, err
	}
	if lib.ttsLoadPersona, err = mustProc("ttsLoadPersona"); err != nil {
		return nil, err
	}
	if lib.ttsSetAudio, err = mustProc("ttsSetAudio"); err != nil {
		return nil, err
	}
	if lib.ttsRead, err = mustProc("ttsRead"); err != nil {
		return nil, err
	}
	if lib.ttsSetCallback, err = mustProc("ttsSetCallback"); err != nil {
		return nil, err
	}
	if lib.ttsSetSpeed, err = mustProc("ttsSetSpeed"); err != nil {
		return nil, err
	}
	if lib.ttsQuery, err = mustProc("ttsQuery"); err != nil {
		return nil, err
	}
	if lib.ttsGetPCM, err = mustProc("ttsGetPCM"); err != nil {
		return nil, err
	}
	if lib.ttsSetTextEncoding, err = mustProc("ttsSetTextEncoding"); err != nil {
		return nil, err
	}
	if lib.ttsEnableEvent, err = mustProc("ttsEnableEvent"); err != nil {
		return nil, err
	}
	if lib.ttsGetErrorMessage, err = mustProc("ttsGetErrorMessage"); err != nil {
		return nil, err
	}
	if lib.ttsGetVersionInfo, err = mustProc("ttsGetVersionInfo"); err != nil {
		return nil, err
	}

	return lib, nil
}

func (l *TTSLibrary) Close() error {
	l.executor.Close()
	h := l.dll.Handle
	l.dll = nil
	if h == 0 {
		return nil
	}
	return windows.FreeLibrary(h)
}
