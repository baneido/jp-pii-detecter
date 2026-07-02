package normalize

import (
	"testing"

	"github.com/baneido/jp-pii-detector/internal/piifixtures"
)

func TestLine(t *testing.T) {
	piifixtures.Require(t)
	tests := []struct {
		name, in, want string
	}{
		{"全角数字", "０１２３４５６７８９", "0123456789"},
		{"全角英字と記号", "ＡＢｃ＠：＝", "ABc@:="},
		{"全角スペース", piifixtures.MustGet(t, "normalize.name_fullwidth_in"), piifixtures.MustGet(t, "normalize.name_fullwidth_out")},
		{"全角ハイフン", piifixtures.MustGet(t, "normalize.fw_phone_in"), piifixtures.MustGet(t, "normalize.fw_phone_out")},
		{"ハイフン類似文字", piifixtures.MustGet(t, "normalize.hyphen_phone_in"), piifixtures.MustGet(t, "normalize.hyphen_phone_out")},
		{"長音記号が数字に隣接", piifixtures.MustGet(t, "normalize.lv_phone_in"), piifixtures.MustGet(t, "normalize.lv_phone_out")},
		{"カタカナ語の長音記号は保持", "サーバー", "サーバー"},
		{"郵便マークは保持", "〒150-0043", "〒150-0043"},
		{"ASCII はそのまま", "hello world 123", "hello world 123"},
		{"行頭の長音記号と数字", "ー123", "-123"},
		{"行末の数字と長音記号", "123ー", "123-"},
		{"数字に隣接しない長音記号は保持", "データー入力", "データー入力"},
		{"SMALL HYPHEN-MINUS", piifixtures.MustGet(t, "normalize.small_hyphen_phone_in"), piifixtures.MustGet(t, "normalize.small_hyphen_phone_out")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Line(tt.in); got != tt.want {
				t.Errorf("Line(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestLineFoldsHalfwidthKatakana は半角カナ（U+FF61-FF9F）を対応する全角カナ・
// 句読点へ折り畳むことを確認する（issue #58 段階 1）。濁点・半濁点（ﾞﾟ）は
// 結合文字 U+3099/U+309A のまま返る（1 ルーン = 1 ルーンの不変条件を保つため、
// 直前の仮名と合成しない）。
func TestLineFoldsHalfwidthKatakana(t *testing.T) {
	const dakuten = "゙"
	const handakuten = "゚"
	tests := []struct{ name, in, want string }{
		{"半角カナ ア行", "ｱｲｳｴｵ", "アイウエオ"},
		{"半角カナ 拗音・促音", "ｷｬｯﾁ", "キャッチ"},
		{"半角カナ 濁点（未合成のまま）", "ｶﾞｷﾞｸﾞｹﾞｺﾞ", "カ" + dakuten + "キ" + dakuten + "ク" + dakuten + "ケ" + dakuten + "コ" + dakuten},
		{"半角カナ 半濁点（未合成のまま）", "ﾊﾟﾋﾟﾌﾟﾍﾟﾎﾟ", "ハ" + handakuten + "ヒ" + handakuten + "フ" + handakuten + "ヘ" + handakuten + "ホ" + handakuten},
		{"半角句読点・中点", "ｱｲｳ｡ｴｵ､ｶｷ･ｸ｢ｹ｣", "アイウ。エオ、カキ・ク「ケ」"},
		{"半角プロロング記号（数字非隣接はー）", "ｻｰﾊﾞｰ", "サー" + "ハ" + dakuten + "ー"},
		{"フリガナラベル（半角カナ）", "ﾌﾘｶﾞﾅ: ﾔﾏﾀﾞ ﾀﾛｳ", "フリ" + "カ" + dakuten + "ナ: ヤマ" + "タ" + dakuten + " タロウ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Line(tt.in); got != tt.want {
				t.Errorf("Line(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestLineHalfwidthProlongedMarkDigitAdjacency は半角プロロング記号
// （U+FF70）が全角「ー」へ写像された後も、数字隣接時のみ '-' になる既存規則が
// そのまま適用されることを確認する。
func TestLineHalfwidthProlongedMarkDigitAdjacency(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"半角プロロング記号 数字直後", "123ｰ", "123-"},
		{"半角プロロング記号 数字直前", "ｰ123", "-123"},
		// 1 つめの ｰ は前後とも非数字（ー のまま）、2 つめの ｰ は直後が数字（- になる）。
		{"半角プロロング記号 混在", "ｻｰﾊﾞｰ123", "サー" + "ハ" + "゙" + "-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Line(tt.in); got != tt.want {
				t.Errorf("Line(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestLineHalfwidthKatakanaKeepsRuneCount は半角カナを含む行でも 1 ルーン = 1
// ルーンの不変条件（正規化後の位置が元テキストの位置と一致する）が保たれることを
// 確認する。フィクスチャに依存しない。
func TestLineHalfwidthKatakanaKeepsRuneCount(t *testing.T) {
	for _, in := range []string{
		"ﾌﾘｶﾞﾅ: ﾔﾏﾀﾞ ﾀﾛｳ",
		"住所: ﾄｳｷｮｳﾄ",
		"ｱｲｳｴｵｶﾞｷﾞｸﾞｹﾞｺﾞﾊﾟﾋﾟﾌﾟﾍﾟﾎﾟ",
	} {
		got := Line(in)
		if gotLen, wantLen := len([]rune(got)), len([]rune(in)); gotLen != wantLen {
			t.Errorf("Line(%q) rune count = %d, want %d (元テキストと同じ)", in, gotLen, wantLen)
		}
	}
}

// TestLineHalfwidthKatakanaIsNotASCIIFastPath は半角カナを含む行が ASCII
// ファストパスを通らず、実際に変換されることを確認する（変換対象を含むため
// needsConversion が true を返すはず）。
func TestLineHalfwidthKatakanaIsNotASCIIFastPath(t *testing.T) {
	in := "ｻﾄｳ"
	got := Line(in)
	if got == in {
		t.Errorf("Line(%q) = %q, 半角カナが変換されていない", in, got)
	}
	if got != "サトウ" {
		t.Errorf("Line(%q) = %q, want %q", in, got, "サトウ")
	}
}

func TestLineKeepsRuneCount(t *testing.T) {
	piifixtures.Require(t)
	in := piifixtures.MustGet(t, "normalize.postal_addr_in")
	if got, want := len([]rune(Line(in))), len([]rune(in)); got != want {
		t.Errorf("rune count changed: %d != %d", got, want)
	}
}

// 変換不要な行はアロケーションなしで同一文字列を返す（ファストパス）。
func TestLineASCIIFastPathReturnsSameString(t *testing.T) {
	piifixtures.Require(t)
	in := "hello world " + piifixtures.MustGet(t, "normalize.fw_phone_out")
	if got := Line(in); got != in {
		t.Errorf("Line(%q) = %q, want unchanged", in, got)
	}
	if testing.AllocsPerRun(10, func() { Line(in) }) != 0 {
		t.Error("ASCII fast path should not allocate")
	}
}

// 変換対象を含まない通常の日本語行もファストパスで割り当てなしに返す
// （漢字・かな・数字非隣接の長音記号だけの行）。フィクスチャ非依存。
func TestLineJapaneseNoConversionFastPath(t *testing.T) {
	for _, in := range []string{
		"これは普通の日本語の文章です。",
		"サーバーの設定を確認する", // 数字に隣接しない長音記号は保持
		"顧客の連絡先を控える",
	} {
		if got := Line(in); got != in {
			t.Errorf("Line(%q) = %q, want unchanged", in, got)
		}
		if testing.AllocsPerRun(10, func() { Line(in) }) != 0 {
			t.Errorf("変換不要な日本語行は割り当てなしで返すべき: %q", in)
		}
	}
}

func BenchmarkLineJapaneseNoConversion(b *testing.B) {
	line := "顧客の氏名と連絡先をサーバーで管理する設定について"
	b.ReportAllocs()
	for b.Loop() {
		Line(line)
	}
}

func BenchmarkLineASCII(b *testing.B) {
	line := `	if err := json.NewEncoder(w).Encode(resp); err != nil { return err }`
	b.ReportAllocs()
	for b.Loop() {
		Line(line)
	}
}

func BenchmarkLineJapanese(b *testing.B) {
	piifixtures.Require(b)
	line := "電話番号：" + piifixtures.MustGet(b, "normalize.fw_lv_phone_bench")
	b.ReportAllocs()
	for b.Loop() {
		Line(line)
	}
}
