package detect

import (
	"strings"

	"github.com/baneido/jp-pii-detector/internal/normalize"
)

const negativeContextWindowRunes = 20

// lineIdx は隣接行相関で検出された finding が乗る行（0 始まり）。前後の
// 論理隣接行（間が空白のみで最大 maxAdjacentLineGap 行差までの非空白行）を
// 見て負コンテキストを判定する。ScanContent の隣接行相関（scanAdjacentLines）が
// 空行を挟んだラベルまで届くようになったのに合わせ、ここも同じ規則で空行を
// スキップしないと、口座番号の直後に空行を挟んだ先に金額の単位（円）が
// 続くようなケースで、負コンテキストによる抑制を取りこぼす。
func (d *Detector) hasCrossLineNegativeContext(f Finding, lines []string, lineIdx int) bool {
	if lineIdx < 0 || lineIdx >= len(lines) {
		return false
	}
	var negCtx []string
	for _, r := range d.rules {
		if r.ID == f.RuleID {
			negCtx = r.NegativeContext
			break
		}
	}
	if len(negCtx) == 0 {
		return false
	}

	var parts []string
	offset := 0
	if p := prevNonBlankIndex(lines, lineIdx, maxAdjacentLineGap); p >= 0 {
		prev := normalize.Line(lines[p])
		parts = append(parts, prev)
		offset = len(prev) + 1 // 改行 1 バイト分
	}
	curr := normalize.Line(lines[lineIdx])
	currRunes := []rune(curr)
	if f.start > len(currRunes) || f.end > len(currRunes) {
		return false
	}
	byteStart := len(string(currRunes[:f.start]))
	byteEnd := len(string(currRunes[:f.end]))
	parts = append(parts, curr)
	if n := nextNonBlankIndex(lines, lineIdx, maxAdjacentLineGap); n >= 0 {
		parts = append(parts, normalize.Line(lines[n]))
	}

	combined := strings.Join(parts, "\n")
	// 隣接行を同一視してチェックするため改行を空白に置き換える。
	// 改行と空白は両方とも 1 バイトなのでオフセットは変わらない。
	combined = strings.ReplaceAll(combined, "\n", " ")
	var runes []rune
	return d.hasNegativeContextNear(combined, offset+byteStart, offset+byteEnd, negativeContextWindowRunes, &runes, negCtx)
}

func (d *Detector) hasNegativeContextNear(s string, start, end, radius int, runes *[]rune, kws []string) bool {
	if *runes == nil {
		*runes = []rune(s)
	}
	rs := *runes
	runeStart := len([]rune(s[:start]))
	runeEnd := runeStart + len([]rune(s[start:end]))

	var generic []string
	for _, kw := range kws {
		switch {
		case isCurrencyPrefix(kw):
			if hasUnitBefore(rs, runeStart, radius, []rune(kw)) {
				return true
			}
		case isCurrencySuffix(kw):
			if hasUnitAfter(rs, runeEnd, radius, []rune(kw), false) {
				return true
			}
		case isCounterSuffix(kw):
			if hasUnitAfter(rs, runeEnd, radius, []rune(kw), true) {
				return true
			}
		default:
			generic = append(generic, kw)
		}
	}
	if len(generic) == 0 {
		return false
	}
	return d.containsAnyContext(contextWindow(s, start, end, radius, runes), generic)
}

// isCurrencyPrefix / isCurrencySuffix / isCounterSuffix は
// rule.digitRuleNegativeContext（internal/rule/builtin.go）の各語を
// 単位の種別に分類する。どれにも該当しない語は hasNegativeContextNear で
// 「汎用」として近傍一致のみで扱う。語リスト（rule 側）に追加した語の
// 単位近接判定を効かせるには、この分類（detect 側）も併せて更新すること。
func isCurrencyPrefix(kw string) bool {
	switch kw {
	case "¥", "￥", "$":
		return true
	}
	return false
}

func isCurrencySuffix(kw string) bool {
	switch kw {
	case "円", "千", "万", "億", "%", "％":
		return true
	}
	return false
}

func isCounterSuffix(kw string) bool {
	switch kw {
	case "人", "名", "件", "個", "回", "点":
		return true
	}
	return false
}

func hasUnitBefore(rs []rune, start, radius int, unit []rune) bool {
	if len(unit) == 0 {
		return false
	}
	i := start - 1
	from := start - radius
	if from < 0 {
		from = 0
	}
	for i >= from && (rs[i] == ' ' || rs[i] == '\t') {
		i--
	}
	unitStart := i - len(unit) + 1
	if unitStart < from {
		return false
	}
	return runesEqual(rs[unitStart:i+1], unit)
}

func hasUnitAfter(rs []rune, end, radius int, unit []rune, requireBoundary bool) bool {
	if len(unit) == 0 {
		return false
	}
	i := end
	to := end + radius
	if to > len(rs) {
		to = len(rs)
	}
	for i < to && (rs[i] == ' ' || rs[i] == '\t') {
		i++
	}
	unitEnd := i + len(unit)
	if unitEnd > to || !runesEqual(rs[i:unitEnd], unit) {
		return false
	}
	return !requireBoundary || unitEnd == len(rs) || !isJapaneseLetter(rs[unitEnd])
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isJapaneseLetter(r rune) bool {
	return (r >= 0x3040 && r <= 0x30ff) || (r >= 0x3400 && r <= 0x9fff)
}
