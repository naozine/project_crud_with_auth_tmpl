// Authentication and WebAuthn functions

// Conditional UI (Passkey autofill) initialization
// ページロード時にパスキーの自動補完を開始

// AbortController for Conditional UI - パスキー登録前に中止するため
let conditionalUIController = null;

document.addEventListener('DOMContentLoaded', function() {
    initConditionalUI();
});

// Conditional UI を中止する関数
function abortConditionalUI() {
    if (conditionalUIController) {
        conditionalUIController.abort();
        conditionalUIController = null;
        console.log('Conditional UI aborted');
    }
}

async function initConditionalUI() {
    // WebAuthn がサポートされているか確認
    if (!window.PublicKeyCredential) {
        console.log('WebAuthn is not supported');
        return;
    }

    // Conditional Mediation がサポートされているか確認
    if (!PublicKeyCredential.isConditionalMediationAvailable) {
        console.log('Conditional Mediation is not available');
        return;
    }

    try {
        const available = await PublicKeyCredential.isConditionalMediationAvailable();
        if (!available) {
            console.log('Conditional Mediation is not available on this browser');
            return;
        }

        console.log('Conditional UI available, starting...');

        // Discoverable 認証のオプションを取得
        const startResp = await fetch('/webauthn/login/discoverable', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({})
        }).then(r => r.json());

        if (startResp.error) {
            console.log('No passkeys registered or error:', startResp.error);
            return;
        }

        // オプションを準備
        const options = startResp.options;
        options.challenge = base64urlToBuffer(options.challenge);

        if (options.allowCredentials) {
            options.allowCredentials = options.allowCredentials.map(c => {
                c.id = base64urlToBuffer(c.id);
                return c;
            });
        }

        // AbortController を作成
        conditionalUIController = new AbortController();

        // Conditional UI でパスキー認証を開始（ユーザーがパスキーを選択するまで待機）
        const assertion = await navigator.credentials.get({
            publicKey: options,
            mediation: 'conditional',  // ← これが Conditional UI のキー
            signal: conditionalUIController.signal
        });

        // パスキーが選択されたら認証を完了
        showAuthMessage("パスキーでログイン中...", false);

        const finishReq = {
            challenge_id: startResp.challenge_id,
            response: {
                id: assertion.id,
                rawId: bufferToBase64url(assertion.rawId),
                type: assertion.type,
                response: {
                    authenticatorData: bufferToBase64url(assertion.response.authenticatorData),
                    clientDataJSON: bufferToBase64url(assertion.response.clientDataJSON),
                    signature: bufferToBase64url(assertion.response.signature),
                    userHandle: assertion.response.userHandle ? bufferToBase64url(assertion.response.userHandle) : null
                }
            }
        };

        const finishResp = await fetch('/webauthn/login/finish', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(finishReq)
        }).then(r => r.json());

        if (finishResp.error) {
            showAuthMessage("ログインエラー: " + finishResp.error, true);
            return;
        }

        if (finishResp.redirect_url) {
            window.location.href = finishResp.redirect_url;
        }

    } catch (e) {
        // ユーザーがキャンセルした場合やエラーの場合
        if (e.name !== 'AbortError' && e.name !== 'NotAllowedError') {
            console.error('Conditional UI error:', e);
        }
    }
}

// Base64URL utility functions
function base64urlToBuffer(base64url) {
    if (!base64url || typeof base64url !== 'string') {
        throw new Error('Invalid base64url input');
    }
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    const paddedBase64 = base64 + '='.repeat((4 - base64.length % 4) % 4);
    const binaryString = atob(paddedBase64);
    const buffer = new ArrayBuffer(binaryString.length);
    const view = new Uint8Array(buffer);
    for (let i = 0; i < binaryString.length; i++) {
        view[i] = binaryString.charCodeAt(i);
    }
    return buffer;
}

function bufferToBase64url(buffer) {
    const bytes = new Uint8Array(buffer);
    let binaryString = '';
    for (let i = 0; i < bytes.length; i++) {
        binaryString += String.fromCharCode(bytes[i]);
    }
    const base64 = btoa(binaryString);
    return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

// HTMX Login Form Handler
document.body.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.elt.id !== 'login-form') return;

    try {
        const resp = JSON.parse(evt.detail.xhr.responseText);
        const msgDiv = document.getElementById('auth-messages');
        if (!msgDiv) return;

        msgDiv.classList.remove('hidden', 'bg-red-50', 'text-red-700', 'bg-green-50', 'text-green-700');

        if (evt.detail.successful) {
            msgDiv.classList.add('bg-green-50', 'text-green-700');
            if (resp.magic_link) {
                msgDiv.innerHTML = '<p class="font-bold mb-2">' + resp.message + '</p>' +
                    '<a href="' + resp.magic_link + '" class="underline break-all hover:text-green-900">ログインリンクをクリックして続行</a>';
            } else {
                msgDiv.textContent = resp.message;
            }
        } else {
            msgDiv.classList.add('bg-red-50', 'text-red-700');
            msgDiv.textContent = resp.error || "エラーが発生しました。";
        }
        msgDiv.classList.remove('hidden');
    } catch (e) {
        console.error("Failed to parse response", e);
    }
});

// Helper to show messages (on login page)
function showAuthMessage(text, isError = false) {
    const el = document.getElementById('auth-messages');
    if (!el) {
        // Fallback for pages without the message container (e.g. dashboard)
        alert(text);
        return;
    }
    el.textContent = text;
    el.className = isError
        ? 'p-4 rounded-md bg-red-50 text-red-700'
        : 'p-4 rounded-md bg-green-50 text-green-700';
    el.classList.remove('hidden');
}

// WebAuthn Functions

async function registerPasskey(email) {
    if (!email) {
        showAuthMessage("メールアドレスが見つかりません。", true);
        return;
    }

    if (!window.MagicLink) {
        console.error("MagicLink JS library not loaded");
        return;
    }

    try {
        // Conditional UI が実行中の場合は中止する（WebAuthn は同時に1つの操作のみ許可）
        abortConditionalUI();

        // If on dashboard, we might use alert for progress
        const isDashboard = !document.getElementById('auth-messages');
        if(isDashboard) {
            if(!confirm("この端末をパスキーとして登録しますか？")) return;
        } else {
            showAuthMessage("パスキーを登録中...", false);
        }

        const res = await MagicLink.register(email);

        if (res.success) {
            // Always reload to reflect the new passkey status from server
            location.reload();
        }
    } catch (e) {
        if (!document.getElementById('auth-messages')) alert("エラー: " + e.message);
        else showAuthMessage("エラー: " + e.message, true);
    }
}

async function loginPasskey() {
    const emailInput = document.getElementById('email');
    if (!emailInput) return;
    const email = emailInput.value;

    if (!email) {
        showAuthMessage("メールアドレスを入力してください。", true);
        return;
    }

    try {
        // Conditional UI が実行中の場合は中止する
        abortConditionalUI();

        showAuthMessage("パスキーでログイン中...", false);
        const res = await MagicLink.login(email);
        if (res.success) {
            // Redirection is now handled by the MagicLink library itself via webauthn.js
            // If you need to perform additional actions before redirect, add them here.
        }
    } catch (e) {
        showAuthMessage("ログインエラー: " + e.message, true);
    }
}

async function loginDiscoverable() {
    try {
        // Conditional UI が実行中の場合は中止する
        abortConditionalUI();

        showAuthMessage("端末認証でログイン中...", false);
        const res = await MagicLink.loginDiscoverable();
        if (res.success) {
            // Redirection is now handled by the MagicLink library itself via webauthn.js
            // If you need to perform additional actions before redirect, add them here.
        }
    } catch (e) {
        showAuthMessage("ログインエラー: " + e.message, true);
    }
}
