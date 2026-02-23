package ffi_wrapper

type (
	TTSHandle               uintptr
	TTSResult               int32
	TTSBool                 uint8
	TTSAudioEncodingType    uint32
	TTSAudioSampleType      uint32
	TTSEventType            uint32
	TTSQueryType            uint32
	TTSCallbackFunctionType func(promptID uint32, eventType TTSEventType, iData uintptr, pUser uintptr) uint32
)

const (
	TTSOk    TTSResult = 0
	TTSFalse TTSBool   = 0
	TTSTrue  TTSBool   = 1
)

const (
	TTSAudioEncTypeLinear TTSAudioEncodingType = iota
	TTSAudioEncTypeALAW
	TTSAudioEncTypeULAW
)

const (
	TTSAudioSampleTypeMono   TTSAudioSampleType = 1
	TTSAudioSampleTypeStereo TTSAudioSampleType = 2
)

const (
	TTSEventAudioStart TTSEventType = iota
	TTSEventEndOfSpeech
	TTSEventLanguagePhoneme
	TTSEventVoicePhoneme
	TTSEventData
	TTSEventText
	TTSEventSentence
	TTSEventBookmark
	TTSEventTag
	TTSEventPause
	TTSEventResume
	TTSEventFreeSpace
	TTSEventNotSent
	TTSEventAudio
	TTSEventVoiceChange
	TTSEventLanguageChange
	TTSEventError
	TTSEventGetLesOut
	TTSEventJump
	TTSEventUnitSelection
	TTSEventParagraph
	TTSEventTextEncoding
	TTSEventStyleChange
	TTSEventPersonaChange
	TTSEventReserved = 100
	TTSEventDebug    = 113
)

const (
	TTSQueryObjectReader TTSQueryType = iota
	TTSQueryObjectVoice
	TTSQueryObjectLanguage
	TTSQueryObjectStyle
)

func ToTTSBool(b bool) TTSBool {
	if b {
		return TTSTrue
	}
	return TTSFalse
}
