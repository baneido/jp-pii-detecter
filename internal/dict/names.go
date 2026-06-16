package dict

import (
	"embed"
	"strings"
)

//go:embed surnames.txt given_names.txt
var namesFS embed.FS

var (
	surnames   = loadNameSet("surnames.txt")
	givenNames = loadNameSet("given_names.txt")
)

func loadNameSet(name string) map[string]bool {
	data, err := namesFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	out := map[string]bool{}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = true
	}
	return out
}

// nameComponentMaxRunes は分割検証で許す姓・名 1 要素あたりの最大ルーン数。
// これを超える長い文字列は人名要素として扱わない（一般名詞・組織名の誤検証防止）。
const nameComponentMaxRunes = 4

// IsSurname は s が収録済みの姓かを返す。
func IsSurname(s string) bool { return surnames[s] }

// IsGivenName は s が収録済みの名かを返す。
func IsGivenName(s string) bool { return givenNames[s] }

// IsPersonName は候補文字列 s が人名らしいかを姓名辞書で検証する。
// 単独の姓・単独の名、または「姓 + 名」に分割できる場合に true を返す。
// 空白区切り（"山田 太郎"）と区切りなし（"山田太郎"）の両方に対応する。
//
// この関数は全文走査の検出器ではなく、ラベル・敬称などで得た候補の
// 検証器（validator）として使う想定。辞書は頻出名に絞っているため、
// 収録外の人名は false になりうる（再現率より適合率を優先する設計）。
func IsPersonName(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if surnames[s] || givenNames[s] {
		return true
	}
	// 空白区切りがあれば「姓 + 名」の 2 要素としてのみ検証する。
	if strings.ContainsAny(s, " 　") {
		fields := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '　' })
		return len(fields) == 2 && surnames[fields[0]] && givenNames[fields[1]]
	}
	// 区切りなしは全分割を試し、姓 + 名に分けられれば人名とみなす。
	rs := []rune(s)
	for i := 1; i < len(rs); i++ {
		if i > nameComponentMaxRunes || len(rs)-i > nameComponentMaxRunes {
			continue
		}
		if surnames[string(rs[:i])] && givenNames[string(rs[i:])] {
			return true
		}
	}
	return false
}
