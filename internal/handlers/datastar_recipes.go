package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/naozine/project_crud_with_auth_tmpl/web/components"
)

// datastar_recipes.go は /datastar/recipes（認証不要・本番でも公開）のバックエンド。
// docs/datastar/datastar-llm-guide.md のお手本となる「動くレシピ集」を提供する。
//
// 安全性: DB には一切触れず、状態はプロセス内インメモリのみ（再起動で消える）。
// 本番公開でも悪用でデータが壊れないようにしている。状態は全訪問者で共有されるため
// mutex で保護し、TODO は件数に上限を設け、reset で初期化できる。

const recipeTodoMax = 20

// recipeStore はレシピ集デモ用のインメモリ状態（全訪問者で共有）。
type recipeStore struct {
	mu      sync.Mutex
	counter int
	todos   []components.RecipeTodo
	nextID  int
	items   []components.RecipeItem
}

var recipeState = &recipeStore{}

func (s *recipeStore) snapshotTodos() []components.RecipeTodo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]components.RecipeTodo, len(s.todos))
	copy(out, s.todos)
	return out
}

func (s *recipeStore) incr() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return s.counter
}

func (s *recipeStore) addTodo(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.todos = append(s.todos, components.RecipeTodo{ID: s.nextID, Text: text})
	if len(s.todos) > recipeTodoMax {
		s.todos = s.todos[len(s.todos)-recipeTodoMax:]
	}
}

func (s *recipeStore) removeTodo(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.todos[:0]
	for _, t := range s.todos {
		if t.ID != id {
			out = append(out, t)
		}
	}
	s.todos = out
}

func (s *recipeStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter = 0
	s.todos = nil
	s.items = nil
}

// items はダイアログ編集デモ用。初回アクセス時に seed し、全訪問者で共有、
// reset で初期化される。
func (s *recipeStore) seedItemsLocked() {
	if len(s.items) == 0 {
		s.items = []components.RecipeItem{
			{ID: 1, Name: "Alpha"},
			{ID: 2, Name: "Bravo"},
			{ID: 3, Name: "Charlie"},
		}
	}
}

func (s *recipeStore) snapshotItems() []components.RecipeItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedItemsLocked()
	out := make([]components.RecipeItem, len(s.items))
	copy(out, s.items)
	return out
}

func (s *recipeStore) item(id int) (components.RecipeItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedItemsLocked()
	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return components.RecipeItem{}, false
}

func (s *recipeStore) updateItem(id int, name string) (components.RecipeItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedItemsLocked()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Name = name
			return s.items[i], true
		}
	}
	return components.RecipeItem{}, false
}

// recipeFruits はライブ検索デモ用の固定リスト。
var recipeFruits = []string{
	"apple", "apricot", "banana", "blueberry", "cherry", "grape", "lemon",
	"mango", "orange", "peach", "pear", "pineapple", "strawberry", "watermelon",
}

// DatastarRecipesPage はレシピ集ページ本体を描画する（認証不要）。
func DatastarRecipesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	_ = components.DatastarRecipes(recipeState.snapshotTodos(), recipeState.snapshotItems()).Render(r.Context(), w)
}

// --- レシピ 2: サーバ往復カウンタ（@post → MarshalAndPatchSignals）---

func RecipeCounterInc(w http.ResponseWriter, r *http.Request) {
	count := recipeState.incr()
	sse := datastar.NewSSE(w, r)
	_ = sse.MarshalAndPatchSignals(map[string]any{"count": count})
}

// --- レシピ 3: インメモリ TODO 追加（ReadSignals → PatchElementTempl でリスト再描画）---

func RecipeTodoAdd(w http.ResponseWriter, r *http.Request) {
	var sig struct {
		Text string `json:"text"`
	}
	if err := datastar.ReadSignals(r, &sig); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if t := strings.TrimSpace(sig.Text); t != "" {
		recipeState.addTodo(t)
	}
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(
		components.RecipeTodoList(recipeState.snapshotTodos()),
		datastar.WithSelectorID("recipe-todos"),
		datastar.WithModeInner(),
	)
	// 入力欄をクリア
	_ = sse.MarshalAndPatchSignals(map[string]any{"text": ""})
}

// --- レシピ 4: TODO 削除（RemoveElementByID は使わずリスト再描画で一貫させる）---

func RecipeTodoRemove(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	recipeState.removeTodo(id)
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(
		components.RecipeTodoList(recipeState.snapshotTodos()),
		datastar.WithSelectorID("recipe-todos"),
		datastar.WithModeInner(),
	)
}

// --- レシピ 5: ライブ検索（data-on:input__debounce → @get → PatchElementTempl）---

func RecipeSearch(w http.ResponseWriter, r *http.Request) {
	var sig struct {
		Query string `json:"query"`
	}
	_ = datastar.ReadSignals(r, &sig)
	q := strings.ToLower(strings.TrimSpace(sig.Query))
	hits := make([]string, 0, len(recipeFruits))
	for _, f := range recipeFruits {
		if q == "" || strings.Contains(f, q) {
			hits = append(hits, f)
		}
	}
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(
		components.RecipeSearchResults(hits),
		datastar.WithSelectorID("recipe-search-results"),
		datastar.WithModeInner(),
	)
}

// --- レシピ 6: indicator/ローディング（わざと遅延 → data-indicator が反応）---

func RecipeSlow(w http.ResponseWriter, r *http.Request) {
	time.Sleep(1200 * time.Millisecond)
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElements(
		`<p id="recipe-slow-result" class="text-green-700">完了しました（サーバ側で1.2秒待機）</p>`,
		datastar.WithModeOuter(),
	)
}

// --- レシピ 7: ポーリング（data-on-interval → @get → PatchElements でサーバ時刻）---

func RecipeTick(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Format("15:04:05")
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElements(
		fmt.Sprintf(`<span id="recipe-tick" class="font-mono">%s</span>`, now),
		datastar.WithModeOuter(),
	)
}

// --- レシピ 8: ダイアログ（PatchElementTempl で挿入 → ExecuteScript で showModal）---

func RecipeDialog(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(
		components.RecipeDialog(),
		datastar.WithSelectorID("recipe-dialog-container"),
		datastar.WithModeInner(),
	)
	_ = sse.ExecuteScript("document.getElementById('recipe-dialog')?.showModal()")
}

// --- レシピ 9: 仮想スクロール（JS 不要・サーバ往復型）---
// 可視範囲＋overscan の行だけを translateY 付きで差し替える。DOM 上の行は常に一定。

func RecipeVRows(w http.ResponseWriter, r *http.Request) {
	var sig struct {
		Vstart int `json:"vstart"`
	}
	_ = datastar.ReadSignals(r, &sig)
	start := sig.Vstart - components.RecipeVOverscan
	if start < 0 {
		start = 0
	}
	end := sig.Vstart + components.RecipeVVisible + components.RecipeVOverscan
	if end > components.RecipeVTotal {
		end = components.RecipeVTotal
	}
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(
		components.RecipeVRows(start, end),
		datastar.WithSelectorID("vrows"),
		datastar.WithModeOuter(),
	)
}

// --- レシピ 10: ダイアログ編集（遷移なし・PRG の代替）---
// 行の edit → @get でダイアログ表示。保存(@put)で該当行だけ patch しダイアログを閉じる。
// ページ遷移も reload もしない。

func RecipeItemEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	item, ok := recipeState.item(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	sse := datastar.NewSSE(w, r)
	_ = sse.PatchElementTempl(
		components.RecipeItemEditDialog(item),
		datastar.WithSelectorID("recipe-item-dialog-container"),
		datastar.WithModeInner(),
	)
	_ = sse.ExecuteScript("document.getElementById('recipe-item-dialog')?.showModal()")
}

func RecipeItemUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var sig struct {
		Editname string `json:"editname"`
	}
	_ = datastar.ReadSignals(r, &sig)
	item, ok := recipeState.updateItem(id, strings.TrimSpace(sig.Editname))
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	sse := datastar.NewSSE(w, r)
	// 該当行だけ outer 置換。reload も遷移もしない（Datastar 流の編集）。
	_ = sse.PatchElementTempl(
		components.RecipeItemRow(item),
		datastar.WithSelectorID(fmt.Sprintf("item-%d", id)),
		datastar.WithModeOuter(),
	)
	_ = sse.ExecuteScript("document.getElementById('recipe-item-dialog')?.close()")
}

// --- リセット（インメモリ状態を初期化してリロード）---

func RecipeReset(w http.ResponseWriter, r *http.Request) {
	recipeState.reset()
	sse := datastar.NewSSE(w, r)
	_ = sse.ExecuteScript("window.location.reload()")
}
