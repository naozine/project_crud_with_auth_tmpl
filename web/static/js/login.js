// ログインフォームの処理。
// nz-magic-linkライブラリの/auth/loginエンドポイントはJSONレスポンスを返すため、
// HTMXのHTMLスワップを直接使用できません。
// そのため、ここでは例外的にJavaScriptでJSONレスポンスをハンドリングし、
// DOMを更新しています。
// これは原則である「HTMXによるHTMLスワップ」からの逸脱であり、
// 外部ライブラリの制約によるやむを得ない措置です。
// 通常のHTMLレスポンスを返すエンドポイントでは、この方法は避けてください。
document.body.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.elt.id !== 'login-form') return;

    try {
        const resp = JSON.parse(evt.detail.xhr.responseText);
        const msgDiv = document.getElementById('auth-messages');
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
