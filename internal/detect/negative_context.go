package detect

import (
	"strings"

	"github.com/baneido/jp-pii-detector/internal/normalize"
	"github.com/baneido/jp-pii-detector/internal/rule"
)

const negativeContextWindowRunes = 20

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
	if lineIdx > 0 {
		prev := normalize.Line(lines[lineIdx-1])
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
	if lineIdx+1 < len(lines) {
		parts = append(parts, normalize.Line(lines[lineIdx+1]))
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
		switch rule.ClassifyNegativeKeyword(kw) {
		case rule.NegativeKeywordCurrencyPrefix, rule.NegativeKeywordLabelPrefix:
			// 通貨記号（¥100）と採番ラベル（伝票番号 100...）は、どちらも
			// 値の直前に隣接する場合のみ抑制する（hasUnitBefore）。
			if hasUnitBefore(rs, runeStart, radius, []rune(kw)) {
				return true
			}
		case rule.NegativeKeywordCurrencySuffix:
			if hasUnitAfter(rs, runeEnd, radius, []rune(kw), false) {
				return true
			}
		case rule.NegativeKeywordCounterSuffix:
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
	// requireBoundary はカウンタ接尾語（件・人 等）専用。直後が漢字なら
	// 「件名」「名義」のような漢字複合語の一部とみなし、単位としては
	// 扱わない（境界不成立）。ひらがな（件に/件が/件を のような助詞続き）や
	// 記号・行末は単位として独立しているとみなし、抑制を適用する。
	return !requireBoundary || unitEnd == len(rs) || !isKanji(rs[unitEnd])
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

// isKanji は CJK 統合漢字（拡張 A を含む）かどうかを返す。ひらがな・
// カタカナはここに含めない（hasUnitAfter の requireBoundary が、助詞続き
// （件に/件が 等）と漢字複合語（件名 等）を区別するために使う）。
func isKanji(r rune) bool {
	return r >= 0x3400 && r <= 0x9fff
}
