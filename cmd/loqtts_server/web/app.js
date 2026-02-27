// OpenAI TTS Client main logic

class TTSClient {
    constructor() {
        this.init();
    }

    init() {
        this.changeLanguage('en');
        this.setupEventListeners();
        this.loadSavedApiKey();
        this.applyTranslations();
        this.loadModels().then();
    }


    // Setup event listeners
    setupEventListeners() {
        // Speed slider
        const speedSlider = document.getElementById('speed');
        const speedValue = document.querySelector('.speed-value');
        speedSlider.addEventListener('input', (e) => {
            speedValue.textContent = e.target.value + 'x';
        });

        // API key
        const apiKeyInput = document.getElementById('apiKey');
        apiKeyInput.addEventListener('blur', () => {
            localStorage.setItem('openai_api_key', apiKeyInput.value);
            this.loadModels();
        });

        // Form submission
        document.getElementById('ttsForm').addEventListener('submit', (e) => {
            this.handleFormSubmit(e);
        });

        // Model dropdown
        const modelSelect = document.getElementById('model');
        modelSelect.addEventListener('change', () => {
            const selectedOption = modelSelect.options[modelSelect.selectedIndex];
            const modelId = selectedOption.value;
            const modelName = selectedOption.textContent;
            console.log(`Selected model: ${modelName} (ID: ${modelId})`);
            localStorage.setItem('selected_model', modelId);
        });
    }

    // Load saved API key
    loadSavedApiKey() {
        const savedKey = localStorage.getItem('openai_api_key');
        if (savedKey) {
            document.getElementById('apiKey').value = savedKey;
            this.loadModels();
        }
    }

    // Translation function
    t(key) {
        const keys = key.split('.');
        let value = window.translations[this.currentLanguage];

        for (const k of keys) {
            if (value && typeof value === 'object') {
                value = value[k];
            } else {
                return key;
            }
        }

        return value || key;
    }

    // Change language
    changeLanguage(language) {
        this.currentLanguage = language;
        localStorage.setItem('tts_language', language);
        this.applyTranslations();
    }

    // Apply translations
    applyTranslations() {
        // Translate elements with data-translate
        document.querySelectorAll('[data-translate]').forEach(element => {
            const key = element.getAttribute('data-translate');
            element.textContent = this.t(key);
        });

        // Translate placeholders
        document.querySelectorAll('[data-translate-placeholder]').forEach(element => {
            const key = element.getAttribute('data-translate-placeholder');
            element.placeholder = this.t(key);
        });

        // Update page title
        document.title = this.t('pageTitle');

        // Update HTML language
        const langMap = {ru: 'ru', en: 'en', zh: 'zh-CN'};
        document.documentElement.lang = langMap[this.currentLanguage] || 'en';

        // Update select options
        this.updateSelectOptions();
    }

    // Update select options
    updateSelectOptions() {
        // Update models
        const modelSelect = document.getElementById('model');
        if (modelSelect.options.length === 1) {
            modelSelect.options[0].textContent = this.t('models.loading');
        }

        // Update formats
        const formatSelect = document.getElementById('format');
        Array.from(formatSelect.options).forEach(option => {
            const key = option.getAttribute('data-translate');
            if (key) {
                option.textContent = this.t(key);
            }
        });
    }

    async loadModels() {
        const apiKey = document.getElementById('apiKey').value;
        const modelSelect = document.getElementById('model');

        try {
            modelSelect.innerHTML = `<option value="">${this.t('models.loadingModels')}</option>`;

            const headers = {};
            if (apiKey) {
                headers["Authorization"] = `Bearer ${apiKey}`;
            }

            const response = await fetch('../v1/models', {headers});

            if (!response.ok) {
                // noinspection ExceptionCaughtLocallyJS
                throw new Error(this.t('errors.loadModels'));
            }

            const data = await response.json();
            const ttsModels = data.data;

            modelSelect.innerHTML = '';

            if (ttsModels.length === 0) {
                modelSelect.innerHTML = `<option value="">${this.t('models.notFound')}</option>`;
                return;
            }

            ttsModels.forEach(model => {
                const option = document.createElement('option');
                option.value = model.id;
                option.textContent = model.name;
                modelSelect.appendChild(option);
            });

            const savedModel = localStorage.getItem('selected_model');
            if (savedModel && ttsModels.some(m => m.id === savedModel)) {
                modelSelect.value = savedModel;
            }

        } catch (error) {
            console.error('Error loading models:', error);
            modelSelect.innerHTML = `<option value="">${this.t('models.loadingError')}</option>`;
        }
    }

    // Update progress
    updateProgress(percentage, status, details = '') {
        const progressBar = document.getElementById('progressBar');
        const loadingStatus = document.getElementById('loadingStatus');
        const loadingDetails = document.getElementById('loadingDetails');
        const loadingTitle = document.getElementById('loadingTitle');

        progressBar.style.width = percentage + '%';
        loadingStatus.textContent = status;
        loadingDetails.textContent = percentage + '%';

        if (percentage < 30) {
            loadingTitle.textContent = this.t('loading.preparing');
        } else if (percentage < 70) {
            loadingTitle.textContent = this.t('loading.generating');
        } else {
            loadingTitle.textContent = this.t('loading.finishing');
        }

        if (details) {
            loadingDetails.textContent += ` • ${details}`;
        }
    }


    // Show errors
    showError(message) {
        const errorDiv = document.getElementById('error');
        errorDiv.innerHTML = `<strong>${this.t('errors.title')}</strong> ${message}`;
        errorDiv.style.display = 'block';
        console.error('Showing error to user:', message);
    }

    // Handle form submission
    async handleFormSubmit(e) {
        e.preventDefault();

        const apiKey = document.getElementById('apiKey').value;
        const model = document.getElementById('model').value;
        const text = document.getElementById('text').value;
        const speed = parseFloat(document.getElementById('speed').value);
        const format = document.getElementById('format').value;

        // Validation
        if (!model) {
            this.showError(this.t('errors.selectModel'));
            return;
        }

        if (text.length > 4096) {
            this.showError(this.t('errors.textTooLong'));
            return;
        }

        // Hide previous results
        document.getElementById('result').style.display = 'none';
        document.getElementById('error').style.display = 'none';
        document.getElementById('loading').style.display = 'block';
        document.getElementById('generateBtn').disabled = true;

        this.updateProgress(0, this.t('loading.initialization'), `${text.length} ${this.t('units.characters')}`);

        try {
            const requestData = {
                model: model,
                input: text,
                voice: "",
                response_format: format
            };

            // Add parameters based on model capabilities
            requestData.speed = speed;

            console.log('Sending request to OpenAI:', requestData);
            this.updateProgress(15, this.t('loading.connecting'), `${this.t('units.model')} ${model}`);

            const response = await fetch('../v1/audio/speech', {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${apiKey}`,
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestData)
            });

            console.log('Received API response:', response.status, response.statusText);

            if (!response.ok) {
                let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
                try {
                    const errorData = await response.json();
                    errorMessage = errorData.error?.message || errorMessage;
                    console.error('API error:', errorData);
                } catch (e) {
                    console.error('Could not read error:', e);
                }
                throw new Error(errorMessage);
            }

            this.updateProgress(70, this.t('loading.downloadingAudio'), `${this.t('units.format')} ${format.toUpperCase()}`);

            const audioBlob = await response.blob();
            console.log('Received audio:', audioBlob.size, this.t('units.bytes'));

            // Wait for progress animation to finish
            this.updateProgress(100, this.t('loading.ready'), `${(audioBlob.size / 1024).toFixed(1)} ${this.t('units.kb')}`);

            // Small delay to show 100%
            await new Promise(resolve => setTimeout(resolve, 500));

            const audioUrl = URL.createObjectURL(audioBlob);

            // Show result
            const audioPlayer = document.getElementById('audioPlayer');
            const downloadBtn = document.getElementById('downloadBtn');
            const resultDiv = document.getElementById('result');

            audioPlayer.src = audioUrl;
            downloadBtn.href = audioUrl;
            downloadBtn.download = `${model}_${Date.now()}.${format}`;

            resultDiv.style.display = 'block';
            resultDiv.classList.add('success-animation');

            console.log('Audio generated successfully!');

        } catch (error) {
            console.error('Generation error:', error);
            this.showError(error.message);
        } finally {
            document.getElementById('loading').style.display = 'none';
            document.getElementById('generateBtn').disabled = false;
        }
    }
}

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
    window.ttsClient = new TTSClient();
});
