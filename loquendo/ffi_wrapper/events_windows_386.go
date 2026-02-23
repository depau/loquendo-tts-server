package ffi_wrapper

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func GetTTSEventDesc(eventType TTSEventType) string {
	switch eventType {
	case TTSEventAudioStart:
		return "TTSEVT_AUDIOSTART: Audio has started flowing"
	case TTSEventEndOfSpeech:
		return "TTSEVT_ENDOFSPEECH: Text-to-Speech conversion has finished"
	case TTSEventLanguagePhoneme:
		return "TTSEVT_LANGUAGEPHONEME: A language phoneme has been produced"
	case TTSEventVoicePhoneme:
		return "TTSEVT_VOICEPHONEME: A voice phoneme has been produced"
	case TTSEventData:
		return "TTSEVT_DATA: Audio rendering of a phoneme has been produced"
	case TTSEventText:
		return "TTSEVT_TEXT: Input text parsing has started"
	case TTSEventSentence:
		return "TTSEVT_SENTENCE: A sentence boundary has been found"
	case TTSEventBookmark:
		return "TTSEVT_BOOKMARK: A text embedded bookmark has been found"
	case TTSEventTag:
		return "TTSEVT_TAG: A text-embedded control tag has been found"
	case TTSEventPause:
		return "TTSEVT_PAUSE: Text-to-Speech conversion has been paused"
	case TTSEventResume:
		return "TTSEVT_RESUME: Text-to-Speech conversion has been resumed"
	case TTSEventFreeSpace:
		return "TTSEVT_FREESPACE: The audio device can receive audio samples"
	case TTSEventNotSent:
		return "TTSEVT_NOTSENT: The audio device is busy"
	case TTSEventAudio:
		return "TTSEVT_AUDIO: An audio tag has been found"
	case TTSEventVoiceChange:
		return "TTSEVT_VOICECHANGE: A voice change tag has been found"
	case TTSEventLanguageChange:
		return "TTSEVT_LANGUAGECHANGE: A language change tag has been found"
	case TTSEventError:
		return "TTSEVT_ERROR: An asynchronous error has occurred"
	case TTSEventGetLesOut:
		return "TTSEVT_GETLESOUT"
	case TTSEventJump:
		return "TTSEVT_JUMP: a \"skip\" command has been issued"
	case TTSEventUnitSelection:
		return "TTSEVT_UNITSELECTION"
	case TTSEventParagraph:
		return "TTSEVT_PARAGRAPH: A paragraph boundary has been detected"
	case TTSEventTextEncoding:
		return "TTSEVT_TEXTENCODING: Input text encoding has been detected"
	case TTSEventStyleChange:
		return "TTSEVT_STYLECHANGE: A style change tag has been found"
	case TTSEventPersonaChange:
		return "TTSEVT_PERSONACHANGE: A persona change tag has been found"
	case TTSEventReserved:
		return "TTSEVT_RESERVED"
	case TTSEventDebug:
		return "TTSEVT_DEBUG"
	default:
		return fmt.Sprintf("unknown event type %d", eventType)
	}
}

func TTSDescribeEvent(eventType TTSEventType, iData uintptr) string {
	switch eventType {
	case TTSEventAudioStart, TTSEventEndOfSpeech, TTSEventPause, TTSEventResume, TTSEventFreeSpace, TTSEventNotSent, TTSEventJump:
		// iData is NULL
		return GetTTSEventDesc(eventType)
	case TTSEventText, TTSEventBookmark, TTSEventTag, TTSEventAudio, TTSEventLanguageChange, TTSEventError, TTSEventParagraph, TTSEventTextEncoding, TTSEventStyleChange:
		// iData is a pointer to a null-terminated string
		str := windows.BytePtrToString((*byte)(unsafe.Pointer(iData)))
		return fmt.Sprintf("%s: '%q'", GetTTSEventDesc(eventType), str)
	case TTSEventSentence:
		// iData is an int
		return fmt.Sprintf("%s: %d", GetTTSEventDesc(eventType), iData)
	default:
		// unsupported or something else; print iData as a hex value for debugging
		return fmt.Sprintf("%s: iData=0x%08X", GetTTSEventDesc(eventType), iData)
	}
}
