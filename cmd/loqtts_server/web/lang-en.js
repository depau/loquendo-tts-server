// English language pack
window.translations = window.translations || {};
window.translations.en = {
    // Headers and main elements
    title: "🚉 Loquendo TTS 🚅",
    pageTitle: "Loquendo TTS Client",
    
    // Information blocks
    modelInfo: {
        title: "💡 Voice Information:",
    },
    
    // Forms and fields
    form: {
        apiKey: "🔑 API Key",
        model: "🤖 TTS Voice",
        text: "📝 Text to Speech",
        textPlaceholder: "Enter text to convert to speech...",
        format: "🎧 Audio Format",
        speed: "⚡ Speech Speed",
        generateBtn: "🎤 Generate Speech"
    },
    
    // Model options
    models: {
        loading: "Loading models...",
        enterKey: "Enter API key to load voices",
        loadingModels: "🔄 Loading models...",
        notFound: "❌ TTS voices not found",
        loadingError: "❌ Error loading voices",
        gpt4oMiniDesc: " (🆕 Latest with prompts)",
        tts1HdDesc: " (HD quality)",
        tts1Desc: " (Fast)"
    },

    // Audio formats
    formats: {
        // mp3: "MP3 (default)",
        // opus: "Opus (low latency)",
        // aac: "AAC (YouTube, mobile)",
        // flac: "FLAC (lossless)",
        wav: "WAV (low latency)",
        // pcm: "PCM (raw data)"
    },
    
    // Loading and progress
    loading: {
        preparing: "Preparing request...",
        generating: "🎤 Generating audio...",
        finishing: "✅ Finishing processing...",
        initialization: "Initialization...",
        sendingRequest: "Sending request to Loquendo API...",
        processingText: "Processing text with neural network...",
        synthesizing: "Synthesizing audio file...",
        gettingResult: "Getting result...",
        connecting: "Connecting to Loquendo API...",
        downloadingAudio: "Downloading audio data...",
        ready: "Ready!"
    },
    
    // Results
    result: {
        title: "✅ Audio ready!",
        download: "⬇️ Download audio"
    },
    
    // Errors
    errors: {
        title: "❌ Error:",
        selectModel: "Select a TTS voice",
        textTooLong: "Text too long! Maximum 4096 characters.",
        loadModels: "Failed to load voices"
    },
    
    // Units
    units: {
        characters: "characters",
        kb: "KB",
        bytes: "bytes",
        model: "Model:",
        format: "Format:"
    }
};
