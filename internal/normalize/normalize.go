// Package normalize は日本語テキスト特有の表記ゆれを正規化する。
//
// 正規化はルーン単位の 1:1 変換に限定している。これにより正規化後の
// ルーン位置が常に元テキストのルーン位置と一致し、検出位置の逆引きが
// 不要になる。
package normalize

// hyphens は「-」に正規化するハイフン類似文字。
// 全角ハイフンマイナス (U+FF0D) は ASCII オフセット変換で処理される。
var hyphens = map[rune]bool{
	'‐': true, // ‐ HYPHEN
	'‑': true, // ‑ NON-BREAKING HYPHEN
	'‒': true, // ‒ FIGURE DASH
	'–': true, // – EN DASH
	'—': true, // — EM DASH
	'―': true, // ― HORIZONTAL BAR
	'−': true, // − MINUS SIGN
	'﹣': true, // ﹣ SMALL HYPHEN-MINUS
}

const prolongedSoundMark = 'ー' // ー（長音記号。数字に隣接する場合のみハイフン扱い）

// halfwidthKatakanaStart/End は半角カナブロック（JIS X 0201 カナ）の範囲。
// U+FF61-FF9F はすべて halfwidthKatakanaFold で 1 対 1 に変換できるため、
// 常に変換対象（isConvTarget）とする。
const (
	halfwidthKatakanaStart = 0xFF61
	halfwidthKatakanaEnd   = 0xFF9F
)

// halfwidthKatakanaFold は半角カナ（U+FF61-FF9F）を対応する全角文字へ写像する
// テーブル（インデックス = コードポイント - halfwidthKatakanaStart）。Unicode の
// 互換分解（NFKD）と同じ対応だが、濁点・半濁点（U+FF9E/FF9F）は結合文字
// （U+3099/U+309A）に写像し、直前の仮名と合成しない。NFKC 相当の合成（ｶﾞ 2ルーン
// → ガ 1ルーン）はルーン数を変えてしまい、internal/normalize の 1 ルーン = 1 ルーンの
// 位置不変条件（正規化後の位置が元テキストの位置と一致する）を破るため、意図的に
// 未合成のまま返す。合成が必要な照合（辞書引きなど）は internal/dict.ComposeKana を
// 呼び出し側で使う。
var halfwidthKatakanaFold = [halfwidthKatakanaEnd - halfwidthKatakanaStart + 1]rune{
	'。', '「', '」', '、', '・', // U+FF61-FF65 句読点・中点
	'ヲ',                                         // U+FF66
	'ァ', 'ィ', 'ゥ', 'ェ', 'ォ', 'ャ', 'ュ', 'ョ', 'ッ', // U+FF67-FF6F 小書き
	'ー',                     // U+FF70 半角プロロング記号
	'ア', 'イ', 'ウ', 'エ', 'オ', // U+FF71-FF75
	'カ', 'キ', 'ク', 'ケ', 'コ', // U+FF76-FF7A
	'サ', 'シ', 'ス', 'セ', 'ソ', // U+FF7B-FF7F
	'タ', 'チ', 'ツ', 'テ', 'ト', // U+FF80-FF84
	'ナ', 'ニ', 'ヌ', 'ネ', 'ノ', // U+FF85-FF89
	'ハ', 'ヒ', 'フ', 'ヘ', 'ホ', // U+FF8A-FF8E
	'マ', 'ミ', 'ム', 'メ', 'モ', // U+FF8F-FF93
	'ヤ', 'ユ', 'ヨ', // U+FF94-FF96
	'ラ', 'リ', 'ル', 'レ', 'ロ', // U+FF97-FF9B
	'ワ', 'ン', // U+FF9C-FF9D
	'゙', '゚', // U+FF9E-FF9F 濁点・半濁点（結合文字。未合成）
}

func mapRune(r rune) rune {
	switch {
	case r >= '！' && r <= '～': // 全角 ASCII → 半角
		return r - 0xFEE0
	case r == '　': // 全角スペース
		return ' '
	case hyphens[r]:
		return '-'
	case r >= halfwidthKatakanaStart && r <= halfwidthKatakanaEnd: // 半角カナ → 全角
		return halfwidthKatakanaFold[r-halfwidthKatakanaStart]
	}
	return r
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// isConvTarget は mapRune が別の文字へ写像する文字（全角 ASCII・全角スペース・
// ハイフン類・半角カナ）かを返す。長音記号「ー」は数字隣接時のみ変換するため、
// ここには含めず needsConversion 側で隣接判定する（半角プロロング記号 U+FF70 は
// 全角「ー」へ無条件変換したうえで、写像後の値に対して同じ隣接判定を適用する）。
func isConvTarget(r rune) bool {
	return (r >= '！' && r <= '～') || r == '　' || hyphens[r] ||
		(r >= halfwidthKatakanaStart && r <= halfwidthKatakanaEnd)
}

// needsConversion は s に変換対象が 1 つでも含まれるかを 1 パスで判定する
// （割り当てなし）。全角 ASCII・全角スペース・ハイフン類のいずれか、または
// 数字に隣接する長音記号があれば true。漢字・かな・数字非隣接の長音記号だけの
// 行（通常の日本語文）は false となり、Line のファストパスで元文字列を返せる。
//
// 旧実装は「U+2010 以上の文字があれば変換が要る」と広く判定していたため、
// 漢字・かな（いずれも U+2010 以上）を含むほぼ全ての日本語行が遅いパスへ入り、
// 変換が不要でも []rune を 2 本割り当てていた。
func needsConversion(s string) bool {
	prev := rune(-1)
	for _, r := range s {
		switch {
		case isConvTarget(r):
			return true
		case r == prolongedSoundMark && isDigit(prev):
			return true
		case isDigit(r) && prev == prolongedSoundMark:
			return true
		}
		prev = r
	}
	return false
}

// Line は 1 行を正規化する。ルーン数は変化しない。
//   - 全角英数字・記号 → 半角
//   - 全角スペース → 半角スペース
//   - ハイフン類似文字 → '-'
//   - 長音記号「ー」は数字に隣接する場合のみ '-'（カタカナ語は保持）
//   - 半角カナ（U+FF61-FF9F）→ 対応する全角カナ・句読点（濁点・半濁点は
//     結合文字 U+3099/U+309A のまま。1 ルーン = 1 ルーンを保つため合成しない）
func Line(s string) string {
	// 変換対象を厳密に判定する。対象がなければ（純 ASCII でも、変換対象を
	// 含まない通常の日本語文でも）割り当てなしで元文字列をそのまま返す。
	if !needsConversion(s) {
		return s
	}
	// 変換が必要な場合のみ []rune を 1 回だけ確保し、その場で書き換える。
	// 入力用と出力用に 2 本のルーン列を持たない（割り当てを 2→1 に削減）。
	rs := []rune(s)
	for i, r := range rs {
		rs[i] = mapRune(r)
	}
	// 長音記号の数字隣接判定は写像後の値で行う。mapRune は「ー」を変えない
	// ため写像後も位置はそのまま残り、全角数字は既に半角化済みである。
	for i, r := range rs {
		if r != prolongedSoundMark {
			continue
		}
		prevDigit := i > 0 && isDigit(rs[i-1])
		nextDigit := i+1 < len(rs) && isDigit(rs[i+1])
		if prevDigit || nextDigit {
			rs[i] = '-'
		}
	}
	return string(rs)
}
