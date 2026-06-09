package components

// recipe_data.go はレシピ集（/datastar/recipes）で使う型と、画面に表示する
// 「お手本コード」の文字列を持つ。文字列はデモの実 markup / 実ハンドラと一致させ、
// 学習者・LLM がそのままコピーできるようにしている。

// RecipeTodo はインメモリ TODO デモの1項目。
type RecipeTodo struct {
	ID   int
	Text string
}

// RecipeItem はダイアログ編集デモ（遷移なし）の1項目。
type RecipeItem struct {
	ID   int
	Name string
}

// 1. 双方向バインド（フロントのみ）
const RecipeBindFront = `<div data-signals:name="''">
  <input data-bind:name placeholder="your name"/>
  <p>Hello, <span data-text="$name"></span></p>
</div>`

// 2. サーバ往復カウンタ（@post → サーバが signal を patch）
const RecipeCounterFront = `<div data-signals:count="0">
  <button data-on:click="@post('/datastar/recipes/api/counter')">+1</button>
  count: <span data-text="$count"></span>
</div>`

const RecipeCounterBack = `func RecipeCounterInc(w http.ResponseWriter, r *http.Request) {
    count := store.incr() // インメモリ。mutex 保護
    sse := datastar.NewSSE(w, r)
    sse.MarshalAndPatchSignals(map[string]any{"count": count})
}`

// 3. インメモリ TODO（追加=ReadSignals→再描画 / 削除=@delete→再描画）
const RecipeTodoFront = `<div data-signals:text="''">
  <form data-on:submit__prevent="@post('/datastar/recipes/api/todos')">
    <input data-bind:text placeholder="new todo"/>
    <button>add</button>
  </form>
  <ul id="recipe-todos"><!-- サーバが PatchElementTempl(inner) で再描画 --></ul>
</div>`

const RecipeTodoBack = `func RecipeTodoAdd(w http.ResponseWriter, r *http.Request) {
    var sig struct{ Text string ` + "`json:\"text\"`" + ` }
    datastar.ReadSignals(r, &sig)
    store.addTodo(strings.TrimSpace(sig.Text))
    sse := datastar.NewSSE(w, r)
    sse.PatchElementTempl(RecipeTodoList(store.snapshotTodos()),
        datastar.WithSelectorID("recipe-todos"), datastar.WithModeInner())
    sse.MarshalAndPatchSignals(map[string]any{"text": ""}) // 入力クリア
}`

// 4. ライブ検索（input を debounce して @get、サーバが結果を patch）
const RecipeSearchFront = `<div data-signals:query="''">
  <input data-bind:query
    data-on:input__debounce.300ms="@get('/datastar/recipes/api/search')"
    placeholder="search fruit"/>
  <ul id="recipe-search-results"><!-- サーバが inner で再描画 --></ul>
</div>`

const RecipeSearchBack = `func RecipeSearch(w http.ResponseWriter, r *http.Request) {
    var sig struct{ Query string ` + "`json:\"query\"`" + ` }
    datastar.ReadSignals(r, &sig)
    hits := filter(fruits, strings.ToLower(sig.Query))
    sse := datastar.NewSSE(w, r)
    sse.PatchElementTempl(RecipeSearchResults(hits),
        datastar.WithSelectorID("recipe-search-results"), datastar.WithModeInner())
}`

// 5. indicator / ローディング（data-indicator が fetch 中だけ true）
const RecipeIndicatorFront = `<div data-indicator:loading>
  <button data-on:click="@get('/datastar/recipes/api/slow')">slow request</button>
  <span data-show="$loading">loading…</span>
  <p id="recipe-slow-result"></p>
</div>`

const RecipeIndicatorBack = `func RecipeSlow(w http.ResponseWriter, r *http.Request) {
    time.Sleep(1200 * time.Millisecond)
    sse := datastar.NewSSE(w, r)
    sse.PatchElements(` + "`<p id=\"recipe-slow-result\">done</p>`" + `,
        datastar.WithModeOuter())
}`

// 6. ポーリング（data-on-interval で定期 @get、サーバ時刻を patch）
const RecipePollFront = `<div>
  <div data-on-interval__duration.1s="@get('/datastar/recipes/api/tick')"></div>
  server time: <span id="recipe-tick" class="font-mono">--:--:--</span>
</div>`

const RecipePollBack = `func RecipeTick(w http.ResponseWriter, r *http.Request) {
    now := time.Now().Format("15:04:05")
    sse := datastar.NewSSE(w, r)
    sse.PatchElements(fmt.Sprintf(` + "`<span id=\"recipe-tick\">%s</span>`" + `, now),
        datastar.WithModeOuter())
}`

// 7. ダイアログ（サーバが <dialog> を patch して showModal）
const RecipeDialogFront = `<div id="recipe-dialog-container"></div>
<button data-on:click="@get('/datastar/recipes/api/dialog')">open dialog</button>`

const RecipeDialogBack = `func RecipeDialog(w http.ResponseWriter, r *http.Request) {
    sse := datastar.NewSSE(w, r)
    sse.PatchElementTempl(RecipeDialog(),
        datastar.WithSelectorID("recipe-dialog-container"), datastar.WithModeInner())
    sse.ExecuteScript("document.getElementById('recipe-dialog')?.showModal()")
}`

// 8. ドロップダウン（クリックでトグル、外側クリックで閉じる。フロントのみ）
// __outside はトグルボタンを含む外側コンテナに付ける。内側に付けると
// ボタン自身のクリックが「外側」と判定され、開いた瞬間に閉じてしまう。
// 項目は <a href="#"> ではなく button にする（href="#" はページ先頭へ飛ぶ）。
const RecipeDropdownFront = `<div data-signals:open="false" data-signals:choice="'(none)'">
  <div data-on:click__outside="$open = false" class="relative">
    <button data-on:click="$open = !$open">menu</button>
    <div data-show="$open">
      <button data-on:click="$choice = 'item 1'; $open = false">item 1</button>
      <button data-on:click="$choice = 'item 2'; $open = false">item 2</button>
    </div>
  </div>
  <p>selected: <span data-text="$choice"></span></p>
</div>`

// 9. 仮想スクロール（JS 不要・サーバ往復型）。
// 全行のうち可視範囲＋overscan だけを描画し、スペーサーで全体高さを保つ。
const (
	RecipeVTotal    = 1000 // 総行数
	RecipeVRowH     = 30   // 行高(px)
	RecipeVVisible  = 10   // 可視行数（コンテナ高さ 300px / 30px）
	RecipeVOverscan = 5    // 可視範囲の上下に余分に描く行数
)

const RecipeVScrollFront = `<div style="height:300px;overflow-y:auto;position:relative"
     data-signals:vstart="0"
     data-on:scroll__throttle.100ms.trailing="
       Math.floor(evt.target.scrollTop / 30) !== $vstart &&
       ($vstart = Math.floor(evt.target.scrollTop / 30), @get('/datastar/recipes/api/vrows'))">
  <!-- 窓(vstart)が変わった時だけ @get（無駄な再取得を避ける）。
       .trailing でスクロール停止時の最終窓も確実に取得する。 -->
  <!-- スペーサー: 全行ぶんの高さでスクロールバーを全件分に見せる -->
  <div style="height:30000px;position:relative">
    <!-- サーバが可視窓だけを translateY 付きで差し替える -->
    <div id="vrows">...初期窓...</div>
  </div>
</div>`

const RecipeVScrollBack = `func RecipeVRows(w http.ResponseWriter, r *http.Request) {
    var sig struct{ Vstart int ` + "`json:\"vstart\"`" + ` }
    datastar.ReadSignals(r, &sig)
    start := max(sig.Vstart-overscan, 0)
    end := min(sig.Vstart+visible+overscan, total)
    sse := datastar.NewSSE(w, r)
    // #vrows を outer 置換。窓は translateY(start*rowH) で正しい位置に。
    sse.PatchElementTempl(RecipeVRows(start, end),
        datastar.WithSelectorID("vrows"), datastar.WithModeOuter())
}`

// 10. ダイアログ編集（遷移なし・PRG の代替）。
// 行の edit → @get でダイアログ表示。保存(@put)で該当行だけ patch しダイアログを閉じる。
// ページ遷移も reload もしない。
const RecipeItemEditFront = `<ul id="recipe-items">
  <li id="item-1" class="flex gap-2">
    <span>Alpha</span>
    <button data-on:click="@get('/datastar/recipes/api/items/1/edit')">edit</button>
  </li>
  <!-- ...other rows... -->
</ul>
<div id="recipe-item-dialog-container"></div>

<!-- @get で挿入される編集ダイアログ（signal は小文字 editname）: -->
<dialog id="recipe-item-dialog" data-signals="{editname: 'Alpha'}">
  <form data-on:submit__prevent="@put('/datastar/recipes/api/items/1')">
    <input data-bind:editname/>
    <button>save</button>
  </form>
</dialog>`

const RecipeItemEditBack = `// 編集ダイアログを開く（@get）
func RecipeItemEdit(w http.ResponseWriter, r *http.Request) {
    item := store.item(id)
    sse := datastar.NewSSE(w, r)
    sse.PatchElementTempl(RecipeItemEditDialog(item),
        datastar.WithSelectorID("recipe-item-dialog-container"), datastar.WithModeInner())
    sse.ExecuteScript("document.getElementById('recipe-item-dialog').showModal()")
}
// 保存（@put）— 該当行だけ patch、reload も遷移もしない
func RecipeItemUpdate(w http.ResponseWriter, r *http.Request) {
    var sig struct{ Editname string ` + "`json:\"editname\"`" + ` }
    datastar.ReadSignals(r, &sig)
    store.updateItem(id, sig.Editname)
    sse := datastar.NewSSE(w, r)
    sse.PatchElementTempl(RecipeItemRow(store.item(id)),
        datastar.WithSelectorID(fmt.Sprintf("item-%d", id)), datastar.WithModeOuter())
    sse.ExecuteScript("document.getElementById('recipe-item-dialog').close()")
}`

// 11. View Transition でフェード（フロントのみ。v1.0.2 の viewTransitionSelector と
// 同じ View Transition API を data-on の __viewtransition 修飾子で使う）
const RecipeVTFront = `<div data-signals:vtshow="true">
  <!-- __viewtransition でクリックハンドラを View Transition でラップ。
       表示切替が既定クロスフェードでフェードする（CSS で 0.4s に調整済み）。 -->
  <button data-on:click__viewtransition="$vtshow = !$vtshow">toggle</button>
  <div data-show="$vtshow">この要素が View Transition でフェードします</div>
</div>`
