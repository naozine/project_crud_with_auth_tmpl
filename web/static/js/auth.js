// Authentication and WebAuthn functions

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
