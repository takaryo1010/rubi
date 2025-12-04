package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Term は辞書の1つのエントリを表す
type Term struct {
	Term string `yaml:"term"`
	Yomi string `yaml:"yomi"`
	Ref  string `yaml:"ref,omitempty"`
}

// Dictionary は辞書ファイル全体の構造
type Dictionary struct {
	Terms []Term `yaml:"terms"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("使い方: go run sort_dict.go <input_file> [output_file]")
		fmt.Println("")
		fmt.Println("例:")
		fmt.Println("  go run sort_dict.go dict.yaml")
		fmt.Println("  go run sort_dict.go dict.yaml dict_sorted.yaml")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := inputFile
	if len(os.Args) >= 3 {
		outputFile = os.Args[2]
	}

	if err := sortDictFile(inputFile, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1)
	}
}

func sortDictFile(inputFile, outputFile string) error {
	// YAMLファイルを読み込み
	fmt.Printf("読み込み中: %s\n", inputFile)
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("ファイルの読み込みに失敗: %w", err)
	}

	// YAMLをパース
	var dict Dictionary
	if err := yaml.Unmarshal(data, &dict); err != nil {
		return fmt.Errorf("YAMLのパースに失敗: %w", err)
	}

	originalCount := len(dict.Terms)
	fmt.Printf("元の用語数: %d\n", originalCount)

	// termsをソート
	sort.Slice(dict.Terms, func(i, j int) bool {
		termI := dict.Terms[i].Term
		termJ := dict.Terms[j].Term

		// .NETなどの特殊文字で始まるものは最後に配置
		if strings.HasPrefix(termI, ".") && !strings.HasPrefix(termJ, ".") {
			return false
		}
		if !strings.HasPrefix(termI, ".") && strings.HasPrefix(termJ, ".") {
			return true
		}

		// 大文字小文字を区別せずにソート
		return strings.ToLower(termI) < strings.ToLower(termJ)
	})

	// YAMLに変換
	output, err := yaml.Marshal(&dict)
	if err != nil {
		return fmt.Errorf("YAMLへの変換に失敗: %w", err)
	}

	// 各エントリの間に空行を追加
	lines := strings.Split(string(output), "\n")
	var formattedLines []string
	for i, line := range lines {
		// "    - term:" で始まる行の前に空行を追加（最初のエントリを除く）
		if strings.HasPrefix(line, "    - term:") && i > 0 {
			formattedLines = append(formattedLines, "")
		}
		formattedLines = append(formattedLines, line)
	}
	formattedOutput := strings.Join(formattedLines, "\n")

	// ファイルに書き込み
	fmt.Printf("書き込み中: %s\n", outputFile)
	if err := os.WriteFile(outputFile, []byte(formattedOutput), 0644); err != nil {
		return fmt.Errorf("ファイルの書き込みに失敗: %w", err)
	}

	sortedCount := len(dict.Terms)
	fmt.Printf("ソート後の用語数: %d\n", sortedCount)
	fmt.Println("✅ ソート完了！")

	return nil
}
