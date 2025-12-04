# rubi: Markdown用ルビ生成CLIツール

## 概要

`rubi` は、Markdown形式の技術ブログやドキュメントにおいて、読み方が難しい専門用語にHTMLのルビタグ（`<ruby>`）を効率的かつ安全に付与するためのCLIツールです。

**「会議での『これって〇〇読みであってますか？』という確認コストをゼロにする」** をコンセプトに、コミュニケーションの円滑化と心理的安全性の向上を目的としています。

### 主な特徴

-   **Safety First**: コードブロック、インラインコード、リンクのURL、HTMLタグなどの変換すべきでない箇所への誤変換を技術的に防止します。
-   **Explicit Control (マニュアルモード)**: ユーザーが明示的に `:rubi` サフィックスを付与した単語のみを変換します。
-   **Automatic Conversion (スキャンモード)**: ドキュメント全体を走査し、辞書に存在する単語を自動でルビ付きに変換します。初出のみ変換するオプションも利用可能です。
-   **Format Preservation**: Markdownの整形（リフォーマット）は行わず、元の文章のインデントや改行を維持します。
-   **Community Driven Dictionary**: 辞書はGitHub上で管理され、`rubi` コマンドを通じて簡単に更新できます。

## インストール

現在、`rubi`はGoのバイナリとして配布されます。以下の手順でインストールできます。

1.  **Goのインストール**: Go 1.22以上がインストールされていることを確認してください。
2.  **`gh` CLIのインストール**: 辞書のダウンロード機能を利用するために、GitHub CLI (`gh`) のインストールと認証が必要です。
    -   [GitHub CLI のインストール](https://github.com/cli/cli#installation)
    -   `gh auth login` コマンドでGitHubに認証してください。
3.  **`rubi`のビルド**:
    ```bash
    git clone https://github.com/takaryo1010/rubi.git
    cd rubi
    go build -o rubi .
    # パスが通っているディレクトリに移動させると便利です
    # mv rubi /usr/local/bin/
    ```

## 使い方

`rubi` は、Markdownファイルを読み込み、ルビを付与した結果を標準出力に出力します。`-w` オプションを使用すると、ファイルを直接上書きできます。

### 基本コマンド

```bash
rubi [options] <input_file>
```

### オプション一覧

| フラグ         | 短縮形 | 説明                                       | デフォルト     |
| :------------- | :----- | :----------------------------------------- | :------------- |
| `--dictionary` | `-d`   | 辞書ファイルのパスを指定                   | `./dict.yaml`  |
| `--write`      | `-w`   | 入力ファイルを上書き保存する               | `false`        |
| `--scan`       | `-s`   | スキャンモード（自動検索）を有効化         | `false`        |
| `--first-only` |        | スキャンモードで各単語の初出のみを変換する | `false`        |
| `--check`      | `-c`   | 辞書ファイルの構文と重複を検証する         | `false`        |
| `--dry-run`    |        | ファイルを変更せず、変換対象リストを表示   | `false`        |

### マニュアルモード (デフォルト)

単語の末尾に `:rubi` サフィックスを付与すると、辞書に基づいてルビが自動的に付与されます。

**例:**

```markdown
Vite:rubi is a fast build tool.
```

**実行:**

```bash
./rubi example.md
```

**出力:**

```html
<p><ruby>Vite<rt>ヴィート</rt></ruby> is a fast build tool.</p>
```

### スキャンモード (`-s` オプション)

ドキュメント全体を走査し、辞書に存在する単語を自動的にルビ付きHTMLに変換します。

**例:**

```markdown
Vite is a fast build tool. Vite is awesome.
```

**実行:**

```bash
./rubi -s example.md
```

**出力:**

```html
<p><ruby>Vite<rt>ヴィート</rt></ruby> is a fast build tool. <ruby>Vite<rt>ヴィート</rt></ruby> is awesome.</p>
```

#### 初出のみ変換 (`--first-only` オプション)

スキャンモードで `--first-only` フラグを使用すると、各単語の初出のみがルビ付きに変換されます。

**例:**

```bash
./rubi -s --first-only example.md
```

**出力:**

```html
<p><ruby>Vite<rt>ヴィート</rt></ruby> is a fast build tool. Vite is awesome.</p>
```

### 辞書ファイルの検証 (`-c` オプション)

辞書ファイルのYAML構文、必須フィールドの有無、重複エントリーなどを検証します。CI/CDパイプラインでの利用に便利です。

```bash
./rubi -c # デフォルトの dict.yaml を検証
./rubi -c -d my_custom_dict.yaml # 特定の辞書ファイルを検証
```

### ドライランモード (`--dry-run` オプション)

実際にはファイルを変更せず、どのような変換が行われるかを標準エラー出力にログとして表示します。

```bash
./rubi --dry-run example.md # マニュアルモードのドライラン
./rubi -s --dry-run example.md # スキャンモードのドライラン
```

### 辞書の初期化と更新

`rubi` は、GitHubリポジトリから辞書ファイルを初期化・更新するためのサブコマンドを提供します。

#### `rubi init`

GitHubリポジトリから `dict.yaml` をダウンロードし、現在のディレクトリに保存します。

```bash
./rubi init # デフォルトのリポジトリからダウンロード
./rubi init --repo your_owner/your_repo # 特定のリポジトリからダウンロード
./rubi init --overwrite # 既存の dict.yaml を上書き
```

#### `rubi dict update`

GitHubリポジトリから `dict.yaml` をダウンロードし、現在のローカルファイルを上書き更新します。

```bash
./rubi dict update # デフォルトのリポジトリから更新
./rubi dict update --repo your_owner/your_repo # 特定のリポジトリから更新
```

## 辞書ファイル (`dict.yaml`) のフォーマット

辞書ファイルはYAML形式で記述します。

```yaml
terms:
  - term: "Vite"
    yomi: "ヴィート"
    ref: "https://ja.vitejs.dev/" # オプション: 読み方の出典

  - term: "gRPC"
    yomi: "ジーアールピーシー"
    ref: "https://grpc.io/"
```

### 必須フィールド

-   `term`: 変換対象の単語
-   `yomi`: ルビとして付与する読み方

## 除外スコープ (Safety First)

以下のMarkdown要素内にある文字列は、**いかなるモードにおいてもルビ変換の対象外**となります。

-   コードブロック内 (` ```code...``` `)
-   インラインコード内 (`` `code` ``)
-   リンクのURL部分 (`[text](https://do.not.change/here)`)
-   HTMLタグ（HTMLブロックやインラインHTML）内 (`<div>...</div>`, `<span>...</span>`)

## 辞書への貢献 (Community Driven)

`rubi` の辞書はオープンソースデータとしてGitHub上で管理されています。

辞書に不足している単語や誤りを見つけた場合は、[GitHubリポジトリ](https://github.com/takaryo1010/rubi) にPull Requestを送ることで貢献できます。貢献ガイドラインは別途用意される予定です。