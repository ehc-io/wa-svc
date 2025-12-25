package api

import (
	"net/http"
)

// AuthPage serves the authentication web interface.
func (h *Handlers) AuthPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(authPageHTML))
}

const authPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WhatsApp Service - Authentication</title>
    <!-- QR code is now generated server-side -->
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #075e54 0%, #128c7e 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 40px;
            max-width: 450px;
            width: 100%;
            text-align: center;
        }

        .logo {
            width: 80px;
            height: 80px;
            background: #25d366;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 24px;
        }

        .logo svg {
            width: 48px;
            height: 48px;
            fill: white;
        }

        h1 {
            color: #1a1a1a;
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 8px;
        }

        .subtitle {
            color: #667781;
            font-size: 14px;
            margin-bottom: 32px;
            line-height: 1.5;
        }

        .status-badge {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 500;
            margin-bottom: 24px;
        }

        .status-badge.disconnected {
            background: #fee2e2;
            color: #dc2626;
        }

        .status-badge.connecting {
            background: #fef3c7;
            color: #d97706;
        }

        .status-badge.pairing {
            background: #dbeafe;
            color: #2563eb;
        }

        .status-badge.connected {
            background: #dcfce7;
            color: #16a34a;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: currentColor;
        }

        .status-badge.connecting .status-dot,
        .status-badge.pairing .status-dot {
            animation: pulse 1.5s infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.4; }
        }

        #qr-container {
            background: #f8fafc;
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 24px;
            display: none;
        }

        #qr-code {
            margin: 0 auto;
            min-height: 256px;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        #qr-code canvas {
            border-radius: 8px;
        }

        .qr-loading {
            color: #667781;
            font-size: 14px;
        }

        .qr-loading .spinner {
            width: 40px;
            height: 40px;
            border: 3px solid #e2e8f0;
            border-top-color: #25d366;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin: 0 auto 12px;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        .qr-instructions {
            color: #667781;
            font-size: 13px;
            margin-top: 16px;
            line-height: 1.5;
        }

        .btn {
            background: #25d366;
            color: white;
            border: none;
            padding: 14px 32px;
            font-size: 16px;
            font-weight: 600;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s;
            width: 100%;
        }

        .btn:hover {
            background: #20bd5a;
            transform: translateY(-1px);
        }

        .btn:active {
            transform: translateY(0);
        }

        .btn:disabled {
            background: #94a3b8;
            cursor: not-allowed;
            transform: none;
        }

        .success-container {
            display: none;
        }

        .success-icon {
            width: 80px;
            height: 80px;
            background: #dcfce7;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 24px;
        }

        .success-icon svg {
            width: 40px;
            height: 40px;
            stroke: #16a34a;
        }

        .success-title {
            color: #16a34a;
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 8px;
        }

        .success-message {
            color: #667781;
            font-size: 14px;
            line-height: 1.5;
        }

        .error-message {
            background: #fee2e2;
            color: #dc2626;
            padding: 12px 16px;
            border-radius: 8px;
            font-size: 14px;
            margin-bottom: 16px;
            display: none;
        }

        .steps {
            text-align: left;
            margin-bottom: 24px;
        }

        .step {
            display: flex;
            gap: 12px;
            margin-bottom: 12px;
            color: #4b5563;
            font-size: 14px;
        }

        .step-number {
            width: 24px;
            height: 24px;
            background: #e2e8f0;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            font-size: 12px;
            flex-shrink: 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
            </svg>
        </div>

        <div id="main-content">
            <h1>Link Your WhatsApp</h1>
            <p class="subtitle">Connect your WhatsApp account to enable the messaging API</p>

            <div id="status-badge" class="status-badge disconnected">
                <span class="status-dot"></span>
                <span id="status-text">Not Connected</span>
            </div>

            <div id="error-message" class="error-message"></div>

            <div id="qr-container">
                <div id="qr-code"></div>
                <p class="qr-instructions">
                    Open WhatsApp on your phone → Settings → Linked Devices → Link a Device → Scan this QR code
                </p>
            </div>

            <div id="steps" class="steps">
                <div class="step">
                    <span class="step-number">1</span>
                    <span>Open WhatsApp on your phone</span>
                </div>
                <div class="step">
                    <span class="step-number">2</span>
                    <span>Go to Settings → Linked Devices</span>
                </div>
                <div class="step">
                    <span class="step-number">3</span>
                    <span>Tap "Link a Device" and scan the QR code</span>
                </div>
            </div>

            <button id="link-btn" class="btn" onclick="startAuth()">
                Generate QR Code
            </button>
        </div>

        <div id="success-content" class="success-container">
            <div class="success-icon">
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                </svg>
            </div>
            <h2 class="success-title">Successfully Connected!</h2>
            <p class="success-message">Your WhatsApp account is now linked and ready to use with the API.</p>
        </div>
    </div>

    <script>
        let pollInterval = null;
        let qrDisplayed = false;
        let qrFetchAttempts = 0;
        let lastQRCode = '';
        const MAX_QR_FETCH_ATTEMPTS = 100; // More attempts since QR refreshes every 20s

        async function checkStatus() {
            try {
                const response = await fetch('/auth/status');
                const data = await response.json();
                console.log('Status:', data);
                updateUI(data);
                return data;
            } catch (error) {
                console.error('Status check failed:', error);
                showError('Failed to check status. Please refresh the page.');
                return null;
            }
        }

        function updateUI(status) {
            const badge = document.getElementById('status-badge');
            const statusText = document.getElementById('status-text');
            const btn = document.getElementById('link-btn');
            const qrContainer = document.getElementById('qr-container');
            const mainContent = document.getElementById('main-content');
            const successContent = document.getElementById('success-content');
            const steps = document.getElementById('steps');

            // Remove all status classes
            badge.className = 'status-badge';

            if (status.ready || status.state === 'connected') {
                // Successfully connected
                badge.classList.add('connected');
                statusText.textContent = 'Connected';
                mainContent.style.display = 'none';
                successContent.style.display = 'block';
                stopPolling();
            } else if (status.state === 'pairing' || status.has_qr) {
                badge.classList.add('pairing');
                statusText.textContent = 'Waiting for scan...';
                btn.style.display = 'none';
                steps.style.display = 'none';
                qrContainer.style.display = 'block';

                // Keep fetching QR code (it refreshes every ~20 seconds)
                if (qrFetchAttempts < MAX_QR_FETCH_ATTEMPTS) {
                    fetchQRCode();
                }
            } else if (status.state === 'connecting') {
                badge.classList.add('connecting');
                statusText.textContent = 'Connecting...';
                btn.disabled = true;
                btn.textContent = 'Connecting...';
                qrContainer.style.display = 'block';
                showQRLoading();
            } else {
                badge.classList.add('disconnected');
                statusText.textContent = 'Not Connected';
                btn.disabled = false;
                btn.textContent = 'Generate QR Code';
                btn.style.display = 'block';
                steps.style.display = 'block';
                qrContainer.style.display = 'none';
                qrDisplayed = false;
                qrFetchAttempts = 0;
            }

            if (status.error && status.state !== 'pairing' && status.state !== 'connecting') {
                showError(status.error);
            }
        }

        function showQRLoading() {
            const container = document.getElementById('qr-code');
            if (!qrDisplayed) {
                container.innerHTML = '<div class="qr-loading"><div class="spinner"></div>Generating QR Code...</div>';
            }
        }

        async function fetchQRCode() {
            qrFetchAttempts++;
            console.log('Fetching QR code, attempt:', qrFetchAttempts);

            try {
                const response = await fetch('/auth/qr');
                const data = await response.json();
                console.log('QR response:', data);

                if (data.qr_image) {
                    // Check if this is a new QR code (they refresh every ~20 seconds)
                    if (data.qr_code !== lastQRCode) {
                        console.log('New QR code received');
                        lastQRCode = data.qr_code;
                        displayQRImage(data.qr_image);
                    }
                    qrDisplayed = true;
                    hideError();
                } else if (data.error) {
                    console.log('QR not ready:', data.error);
                    if (!qrDisplayed) {
                        showQRLoading();
                    }
                } else {
                    if (!qrDisplayed) {
                        showQRLoading();
                    }
                }
            } catch (error) {
                console.error('QR fetch failed:', error);
                if (!qrDisplayed) {
                    showQRLoading();
                }
            }
        }

        function displayQRImage(imageDataUrl) {
            const container = document.getElementById('qr-code');
            console.log('Displaying QR image');

            const img = document.createElement('img');
            img.src = imageDataUrl;
            img.alt = 'WhatsApp QR Code';
            img.style.width = '256px';
            img.style.height = '256px';
            img.style.borderRadius = '8px';

            container.innerHTML = '';
            container.appendChild(img);
        }

        async function startAuth() {
            const btn = document.getElementById('link-btn');
            const qrContainer = document.getElementById('qr-container');
            const steps = document.getElementById('steps');

            btn.disabled = true;
            btn.textContent = 'Initializing...';
            hideError();

            // Show QR container with loading state immediately
            qrContainer.style.display = 'block';
            steps.style.display = 'none';
            showQRLoading();

            try {
                const response = await fetch('/auth/init', { method: 'POST' });
                const data = await response.json();
                console.log('Auth init response:', data);

                if (!response.ok) {
                    throw new Error(data.error || 'Failed to initialize authentication');
                }

                // Start polling for status updates
                startPolling();

            } catch (error) {
                console.error('Auth init failed:', error);
                showError(error.message);
                btn.disabled = false;
                btn.textContent = 'Generate QR Code';
                btn.style.display = 'block';
                steps.style.display = 'block';
                qrContainer.style.display = 'none';
            }
        }

        function startPolling() {
            if (pollInterval) return;

            // Poll every 1 second for faster QR code detection
            pollInterval = setInterval(checkStatus, 1000);
            // Also check immediately
            checkStatus();
        }

        function stopPolling() {
            if (pollInterval) {
                clearInterval(pollInterval);
                pollInterval = null;
            }
        }

        function showError(message) {
            const errorEl = document.getElementById('error-message');
            errorEl.textContent = message;
            errorEl.style.display = 'block';
        }

        function hideError() {
            document.getElementById('error-message').style.display = 'none';
        }

        // Check status on page load
        document.addEventListener('DOMContentLoaded', async () => {
            const status = await checkStatus();
            if (status && (status.state === 'pairing' || status.state === 'connecting' || status.has_qr)) {
                startPolling();
            }
        });
    </script>
</body>
</html>`
