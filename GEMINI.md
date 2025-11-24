    ## Workflow & Execution Constraints
    - **Do NOT Execute `go run`:** This command runs indefinitely and blocks control. Do not run the server.
    - **Build Verification Only:** Limit actions to code creation and build verification (e.g., `go build`).
    - **User Verification:** The user will handle the actual runtime/operation verification.
    
    ## HTMX & Frontend Interaction Guidelines
    - **Principle: HTML over JSON:** 基本的にHTMXのHTMLスワップ機能を使用し、サーバーサイドでレンダリングされたHTMLフラグメントを返すこと。
    - **Exception for JavaScript (JSON Handling):** 外部ライブラリの制約や極めて限定的なケースで、サーバーがJSONレスポンスを返す必要がある場合に限り、クライアントサイドJavaScriptでJSONレスポンスをハンドリングしDOMを更新することを**例外的に**許容する。その際は、コード内にその経緯と理由を明確にコメントとして残すこと。

    ## Code Style