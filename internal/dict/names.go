package dict

import (
	"embed"
	"strings"
)

//go:embed surnames.txt given_names.txt
var namesFS embed.FS

var (
	surnames   = loadNameSet(namesFS, "surnames.txt")
	givenNames = loadNameSet(namesFS, "given_names.txt")
)

// loadNameSet は fsys に go:embed された name（改行区切り、# 始まりはコメント）を
// 集合として読み込む。姓名辞書とローマ字姓名辞書の両方から共用する。
func loadNameSet(fsys embed.FS, name string) map[string]bool {
	data, err := fsys.ReadFile(name)
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

// SplitsAsFullName は s を「姓 + 名」に分割でき、両要素とも辞書に収録されて
// いるかを返す（単独の姓・単独の名は false）。空白区切り（"山田 太郎"）と
// 区切りなし（"山田太郎"）の両方に対応する。
//
// 注: 全角スペース分岐は防御用。本番経路では normalize.Line が U+3000 を半角
// スペースに畳んでから値が渡るため通常は到達しないが、検証器を正規化前の生入力に
// 対して直接呼ぶ呼び出し元（テスト等）でも正しく動くよう両対応にしている。
func SplitsAsFullName(s string) bool {
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, " 　") {
		fields := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '　' })
		return len(fields) == 2 && surnames[fields[0]] && givenNames[fields[1]]
	}
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

// IsPersonName は候補文字列 s が人名らしいかを姓名辞書で検証する。
// 単独の姓・単独の名、または「姓 + 名」に分割できる場合に true を返す。
//
// この関数は全文走査の検出器ではなく、ラベル・敬称などで得た候補の
// 検証器（validator）として使う想定。辞書は頻出名に絞っているため、
// 収録外の人名は false になりうる（再現率より適合率を優先する設計）。
// 単独 1 文字の名は日常語と衝突しやすいため、ラベル種別で絞り込む
// 呼び出し側（builtin.go の validGivenField 等）では別途長さを制限する。
//
// s は照合前に ComposeKana で濁点・半濁点を合成する。半角カナ由来の
// 「ﾔﾏﾀﾞ」は normalize.Line で「ヤマダ」（ダ = タ + 結合濁点、2 ルーン）に
// 折り畳まれるため、合成しないと辞書（濁点合成済み表記で収録）に一致しない。
func IsPersonName(s string) bool {
	s = ComposeKana(strings.TrimSpace(s))
	if s == "" {
		return false
	}
	return surnames[s] || givenNames[s] || SplitsAsFullName(s)
}

// combiningDakuten・combiningHandakuten は半角カナの濁点・半濁点
// （U+FF9E/U+FF9F）が normalize.Line で折り畳まれた結合文字。1 ルーン = 1 ルーンの
// 不変条件を保つため、正規化は基底の仮名と合成せずこの結合文字のまま残す。
const (
	combiningDakuten    = '゙'
	combiningHandakuten = '゚'
)

// kanaComposition は「基底の仮名 + 結合濁点/半濁点」→ 合成済み 1 文字のテーブル
// （ひらがな・カタカナ計 56 ペア）。golang.org/x/text/unicode/norm への依存を
// 避けるため、濁点・半濁点が付きうる仮名だけを対象にした手動テーブルとして持つ
// （NFC 正規化全体の実装ではない）。
var kanaComposition = map[[2]rune]rune{
	{'う', combiningDakuten}:    'ゔ',
	{'か', combiningDakuten}:    'が',
	{'き', combiningDakuten}:    'ぎ',
	{'く', combiningDakuten}:    'ぐ',
	{'け', combiningDakuten}:    'げ',
	{'こ', combiningDakuten}:    'ご',
	{'さ', combiningDakuten}:    'ざ',
	{'し', combiningDakuten}:    'じ',
	{'す', combiningDakuten}:    'ず',
	{'せ', combiningDakuten}:    'ぜ',
	{'そ', combiningDakuten}:    'ぞ',
	{'た', combiningDakuten}:    'だ',
	{'ち', combiningDakuten}:    'ぢ',
	{'つ', combiningDakuten}:    'づ',
	{'て', combiningDakuten}:    'で',
	{'と', combiningDakuten}:    'ど',
	{'は', combiningDakuten}:    'ば',
	{'ひ', combiningDakuten}:    'び',
	{'ふ', combiningDakuten}:    'ぶ',
	{'へ', combiningDakuten}:    'べ',
	{'ほ', combiningDakuten}:    'ぼ',
	{'は', combiningHandakuten}: 'ぱ',
	{'ひ', combiningHandakuten}: 'ぴ',
	{'ふ', combiningHandakuten}: 'ぷ',
	{'へ', combiningHandakuten}: 'ぺ',
	{'ほ', combiningHandakuten}: 'ぽ',
	{'ウ', combiningDakuten}:    'ヴ',
	{'カ', combiningDakuten}:    'ガ',
	{'キ', combiningDakuten}:    'ギ',
	{'ク', combiningDakuten}:    'グ',
	{'ケ', combiningDakuten}:    'ゲ',
	{'コ', combiningDakuten}:    'ゴ',
	{'サ', combiningDakuten}:    'ザ',
	{'シ', combiningDakuten}:    'ジ',
	{'ス', combiningDakuten}:    'ズ',
	{'セ', combiningDakuten}:    'ゼ',
	{'ソ', combiningDakuten}:    'ゾ',
	{'タ', combiningDakuten}:    'ダ',
	{'チ', combiningDakuten}:    'ヂ',
	{'ツ', combiningDakuten}:    'ヅ',
	{'テ', combiningDakuten}:    'デ',
	{'ト', combiningDakuten}:    'ド',
	{'ハ', combiningDakuten}:    'バ',
	{'ヒ', combiningDakuten}:    'ビ',
	{'フ', combiningDakuten}:    'ブ',
	{'ヘ', combiningDakuten}:    'ベ',
	{'ホ', combiningDakuten}:    'ボ',
	{'ワ', combiningDakuten}:    'ヷ',
	{'ヰ', combiningDakuten}:    'ヸ',
	{'ヱ', combiningDakuten}:    'ヹ',
	{'ヲ', combiningDakuten}:    'ヺ',
	{'ハ', combiningHandakuten}: 'パ',
	{'ヒ', combiningHandakuten}: 'ピ',
	{'フ', combiningHandakuten}: 'プ',
	{'ヘ', combiningHandakuten}: 'ペ',
	{'ホ', combiningHandakuten}: 'ポ',
}

// ComposeKana は s に含まれる「基底の仮名 + 結合濁点/半濁点（U+3099/U+309A）」を
// 対応する濁点・半濁点つきの 1 文字へ合成して返す。半角カナの折り畳み
// （normalize.Line）は 1 ルーン = 1 ルーンの位置不変条件を保つため濁点・半濁点を
// 未合成の結合文字のまま返す。姓名辞書は合成済み表記（ガ・ダ 等）で収録して
// いるため、辞書照合の直前でこの関数を通す。結合文字を含まない入力は
// 割り当てなしでそのまま返す。
func ComposeKana(s string) string {
	if !strings.ContainsRune(s, combiningDakuten) && !strings.ContainsRune(s, combiningHandakuten) {
		return s
	}
	rs := []rune(s)
	out := make([]rune, 0, len(rs))
	for i := 0; i < len(rs); i++ {
		if i+1 < len(rs) {
			if c, ok := kanaComposition[[2]rune{rs[i], rs[i+1]}]; ok {
				out = append(out, c)
				i++
				continue
			}
		}
		out = append(out, rs[i])
	}
	return string(out)
}
