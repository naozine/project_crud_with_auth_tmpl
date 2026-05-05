# Ansible 化の計画メモ

このドキュメントは、現在 Makefile (`vps.mk`) ベースで管理されている VPS デプロイ周りを **Ansible に切り出す** ための引き継ぎメモ。次セッションの Claude Code に渡して進める前提。

## やりたいこと

VPS デプロイの構成管理 (provisioning + deployment) を **Ansible で別管理** にする。現状は本リポジトリの `vps.mk` 内に Make ターゲットとして埋め込まれているが:

- Makefile はビルド/開発タスクと混ざっており見通しが悪い
- 構成管理 (Caddy セットアップ、SSH 鍵配備、ファイアウォール等) と「デプロイ実行」が分離されていない
- 派生プロジェクトを増やすほど、各プロジェクトに同じ vps.mk のコピーが増える
- Ansible なら **冪等性**、**playbook の再利用**、**Vault による secret 管理** が標準で揃う

最終形のイメージ:

```
project_crud_with_auth_tmpl/
├── Makefile              # vps.mk の include を削除（または残しつつ Ansible 呼び出しに）
├── deploy.config.example # 残す（Ansible inventory への変換ヒントとして）
└── docs/dev/ansible-migration-plan.md  # ← これ

別管理（候補）:
ansible-deploy/              # 別リポジトリ or 別ディレクトリ
├── inventory/
│   ├── production
│   └── staging
├── playbooks/
│   ├── provision.yml       # サーバ初期化（Caddy, Docker, ufw, etc.）
│   ├── deploy.yml          # アプリのデプロイ（イメージ pull、コンテナ再起動）
│   └── caddy-setup.yml     # Caddy の sites 配信
├── roles/
│   ├── caddy/
│   ├── docker/
│   └── app/
└── group_vars/
    └── all.yml             # 共通変数
```

## 現状のデプロイ系ファイル一覧

次セッション開始時に Claude が読むべきファイル:

| ファイル | 行数 | 役割 |
|---|---|---|
| `vps.mk` | 159 | VPS 用 Make ターゲット（caddy-setup, docker-deploy, dns-setup 等） |
| `fly.mk` | 243 | fly.io 用 Make ターゲット（**今回の Ansible 化の対象外**、残す） |
| `Dockerfile` | 79 | アプリのコンテナ化 |
| `docker-compose.yaml` | (本番用) | リバプロ含むサービス構成 |
| `docker-compose.dev.yaml` | (開発用) | air 用 |
| `entrypoint.sh` | 22 | コンテナ起動スクリプト |
| `caddy/Caddyfile` | (要確認) | Caddy の設定 |
| `caddy/docker-compose.yaml` | (要確認) | Caddy 単体の compose 定義 |
| `caddy/sites` | (要確認) | サイト別の Caddy 設定 |
| `deploy.config.example` | (要確認) | デプロイ先の接続情報・ドメイン設定 |
| `litestream.yml` | (要確認) | DB バックアップ設定（Ansible 化の影響を受ける可能性） |

`vps.mk` の主要ターゲット:
- `caddy-setup` / `caddy-status` / `caddy-logs` / `caddy-reload`
- `dns-setup`
- `docker-deploy` / `docker-restart` / `docker-remote-logs`

これらが Ansible playbook のどのタスクに対応するかを設計する。

## 検討すべき設計判断

### ① Ansible で何を管理するか

| スコープ | 内容 |
|---|---|
| **provisioning のみ** | Caddy インストール、Docker インストール、ufw、SSH 鍵、ユーザ作成等。デプロイは別ツール (Make / GitHub Actions) |
| **deployment のみ** | アプリのデプロイ（コンテナ更新）だけ Ansible 化。provisioning は手動 |
| **両方** | 上記すべて Ansible で。完全な構成管理 |

→ **両方** が筋。中途半端だと管理が混在する。

### ② 既存の `vps.mk` の扱い

| 案 | メリット | デメリット |
|---|---|---|
| A. 撤去して全て Ansible へ | スッキリ | 短期的に既存運用が壊れる |
| B. 残しつつ Ansible 呼び出しに置き換え | 段階移行可能 | 二重管理のリスク |
| C. 別ファイル (`vps-legacy.mk`) として deprecated 扱い | 履歴を残せる | ファイルが増える |

→ **B** が現実的。`make docker-deploy` が裏で `ansible-playbook deploy.yml` を呼ぶ形に段階移行。

### ③ inventory の管理

| 場所 | secret の扱い |
|---|---|
| **本リポジトリに同梱** | inventory はコミット可、secret は Ansible Vault で暗号化 |
| **別リポジトリ (private)** | secret も含めて完全分離。Vault も併用可 |
| **別リポジトリ (組織アカウント)** | チーム共有、CI 統合が容易 |

→ ユーザーの状況 (個人開発、海外フリーランス案件で派生量産) を踏まえると、**別リポジトリ (private)** が現実的。派生プロジェクトと別リポにすれば inventory の派生差分が独立する。

### ④ 実行環境

| 環境 | メリット | デメリット |
|---|---|---|
| ローカルマシンから実行 | 簡単、すぐ始められる | チームメンバーが増えると環境差が出る |
| Bastion / 専用ホスト | 一元管理、踏み台経由でセキュア | ホスト管理コスト |
| GitHub Actions から実行 | CI 統合、push で自動デプロイ | secret 管理に注意、複雑化 |

→ 個人開発なら **ローカル**、本番運用が安定したら **GitHub Actions** に移行が現実的。

### ⑤ 派生プロジェクトとの関係

| 案 | 説明 |
|---|---|
| **各プロジェクトに同梱** | プロジェクトごとに `ansible/` ディレクトリ |
| **別リポに集約 + プロジェクト別 inventory** | 1 つの ansible-deploy リポで複数プロジェクトを管理 |
| **テンプレ化** | `ansible-deploy-tmpl` を作って派生時にコピー |

→ ユーザーの A 戦略（独立進化 + AI 移植）と整合させるなら、**ansible-deploy-tmpl を作って派生時に AI に複製＋カスタマイズさせる** が筋。

### ⑥ fly.io との使い分け

- fly.io: コンテナ系 PaaS、`fly deploy` で完結
- VPS + Caddy + Docker: 自前運用、Ansible 化の対象

両方とも残す。Ansible 化は **VPS 側のみ**。`fly.mk` には触らない。

### ⑦ Litestream の扱い

- 現状: `litestream.yml` でバックアップ設定、`docker-compose.yaml` で起動
- Ansible 化後: provisioning でセットアップ、デプロイで設定更新

→ Ansible role として `roles/litestream/` を作る価値あり。

## 候補アプローチ

### Phase 1: 別リポジトリ作成 + Provisioning だけ Ansible 化

1. `ansible-deploy-tmpl` リポを新規作成（テンプレリポと並列）
2. `roles/caddy/`, `roles/docker/`, `roles/firewall/` を実装
3. `playbooks/provision.yml` で新規 VPS をセットアップ
4. 既存運用 (`vps.mk` の docker-deploy 等) は維持

メリット: 既存運用を壊さず段階移行できる。

### Phase 2: Deployment も Ansible 化

1. `playbooks/deploy.yml` を実装（Docker image build → push → 本番 pull → restart）
2. `Makefile` の `docker-deploy` ターゲットを `ansible-playbook deploy.yml` に置き換え
3. `deploy.config` の構造を `inventory/production` に移行

メリット: 単一のソースから provisioning も deployment もカバー。

### Phase 3: GitHub Actions 統合 (任意)

1. `master` push 時に Ansible が走る
2. Vault パスフレーズや SSH 鍵は GitHub Secrets に
3. 環境別 (staging / production) に手動承認

メリット: 完全自動化。

### スコープを絞る案

最初から全部やらない。**Phase 1 だけ完了** させて、Phase 2/3 は実運用しながら判断する流れで進める。

## 次セッション開始時の手順

新しい Claude Code セッションを開いて、以下を投げる:

```
docs/dev/ansible-migration-plan.md を読んで、デプロイの Ansible 化を進めたい。
まず Phase 1（別リポジトリ ansible-deploy-tmpl の作成 + provisioning の Ansible 化）から始めよう。

設計判断は以下で進める:
- ① 両方（provisioning + deployment）を最終的に管理、まず Phase 1 で provisioning のみ
- ② 既存 vps.mk は残しつつ段階移行（B 案）
- ③ inventory は別 private リポジトリで管理
- ④ 当面はローカルから実行
- ⑤ ansible-deploy-tmpl を作って派生時に AI 複製
- ⑥ fly.mk は触らない
- ⑦ Litestream も role 化

実装に入る前に、現状の vps.mk / Dockerfile / caddy/ を読んで、何をどの role に切り出すか提案して。
```

このプロンプトで Claude が:
1. このメモを読む
2. 既存ファイルを把握
3. role 設計を提案
4. 合意の上で `ansible-deploy-tmpl` リポの中身を実装

## 参考: Ansible 学習が浅い場合

ユーザーが Ansible に慣れていない場合、最初に新セッションで以下を聞くのも手:

```
Ansible の基本構造（playbook / role / inventory / Vault）を、
このプロジェクトの vps.mk と対応付けて簡潔に解説して。
```

これで設計判断の前提が揃う。

## 関連ファイル (このリポジトリ内)

- 本メモ: `docs/dev/ansible-migration-plan.md` ← 引き継ぎドキュメント
- `docs/migrations/README.md`: A 戦略の改善履歴インデックス（テンプレ追従用）
- `README_TEMPLATE.md`: テンプレ運用方針
- `CLAUDE.md`: 作業ルール
