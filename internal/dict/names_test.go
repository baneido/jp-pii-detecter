package dict

import (
	"iter"
	"strings"
	"testing"
)

func mustRead(name string) string {
	data, err := namesFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// splitLines は loadNameSet と同じ規則（# 行・空行を除く）で有効な行を列挙する。
func splitLines(raw string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for line := range strings.SplitSeq(raw, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if !yield(line) {
				return
			}
		}
	}
}

// TestNameDictIntegrity は姓名辞書の整合性を保証する（ファイル内重複・
// 姓名の相互重複がないこと）。macOS の BSD comm/sort/uniq は CJK で誤検出
// するため、シェルではなく Go で確実に検査する。
func TestNameDictIntegrity(t *testing.T) {
	dupCheck := func(name, raw string) {
		// map は重複を吸収するため、生データを再走査して重複行を検出する。
		seen := map[string]bool{}
		for line := range splitLines(raw) {
			if seen[line] {
				t.Errorf("%s に重複エントリ: %q", name, line)
			}
			seen[line] = true
		}
	}
	dupCheck("surnames.txt", mustRead("surnames.txt"))
	dupCheck("given_names.txt", mustRead("given_names.txt"))

	for s := range surnames {
		if givenNames[s] {
			t.Errorf("%q が姓・名の両方に収録されている（どちらかに統一すること）", s)
		}
	}
}

func TestIsSurnameAndGivenName(t *testing.T) {
	if !IsSurname("山田") {
		t.Errorf("IsSurname(山田) = false, want true")
	}
	if IsSurname("太郎") {
		t.Errorf("IsSurname(太郎) = true, want false")
	}
	if !IsGivenName("太郎") {
		t.Errorf("IsGivenName(太郎) = false, want true")
	}
	if IsGivenName("山田") {
		t.Errorf("IsGivenName(山田) = true, want false")
	}
}

func TestIsPersonName(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		// 単独の姓・名
		{"山田", true},
		{"高橋", true},
		{"太郎", true},
		{"花子", true},
		// 姓 + 名（区切りなし）
		{"山田太郎", true},
		{"高橋健太", true},
		{"佐藤花子", true},
		// 姓 + 名（空白区切り）
		{"山田 太郎", true},
		{"佐藤　花子", true}, // 全角スペース
		// 非人名（組織・一般名詞）
		{"田中商事", false},
		{"山田商事株式会社", false},
		{"一覧", false},
		{"重要", false},
		{"", false},
		// 3 要素以上の空白区切りは不可
		{"山田 太郎 様", false},
		// 名 + 姓 の順は不可（姓 + 名のみ）
		{"太郎山田", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := IsPersonName(tt.in); got != tt.want {
				t.Errorf("IsPersonName(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandedNameDictionaryExamples(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"一ノ瀬", true},
		{"越智", true},
		{"凪沙", true},
		{"伊織", true},
		{"越智凪沙", true},
		{"一ノ瀬 伊織", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := IsPersonName(tt.in); got != tt.want {
				t.Errorf("IsPersonName(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestKatakanaNameDictionaryExamples はカタカナ読みの姓・名（internal/dict/gen-names
// で生成、issue #58）が辞書に収録されていることを確認する。
func TestKatakanaNameDictionaryExamples(t *testing.T) {
	if !IsSurname("サトウ") {
		t.Error(`IsSurname("サトウ") = false, want true`)
	}
	if !IsGivenName("サクラ") {
		t.Error(`IsGivenName("サクラ") = false, want true`)
	}
	if IsSurname("サクラ") {
		t.Error(`IsSurname("サクラ") = true, want false（名と同形の姓は無いはず）`)
	}
	if IsGivenName("サトウ") {
		t.Error(`IsGivenName("サトウ") = true, want false（姓と同形の名は除外されているはず）`)
	}
	if !IsPersonName("サトウ サクラ") {
		t.Error(`IsPersonName("サトウ サクラ") = false, want true`)
	}
	if !IsPersonName("サトウサクラ") {
		t.Error(`IsPersonName("サトウサクラ") = false, want true（区切りなし）`)
	}
}

// TestFourCharacterSurname は 4 文字姓（issue #58 で人手追加）が辞書に
// 収録されていることを確認する。従来の辞書は最長 3 文字だった。
func TestFourCharacterSurname(t *testing.T) {
	for _, s := range []string{"勅使河原", "小比類巻", "テシガハラ"} {
		if !IsSurname(s) {
			t.Errorf("IsSurname(%q) = false, want true", s)
		}
	}
}

func TestIsRomajiSurnameAndGivenName(t *testing.T) {
	if !IsRomajiSurname("yamada") {
		t.Error(`IsRomajiSurname("yamada") = false, want true`)
	}
	if IsRomajiSurname("YAMADA") {
		t.Error(`IsRomajiSurname("YAMADA") = true, want false（呼び出し側で小文字化する契約）`)
	}
	if !IsRomajiGivenName("tarou") {
		t.Error(`IsRomajiGivenName("tarou") = false, want true`)
	}
	if IsRomajiSurname("notaname") || IsRomajiGivenName("notaname") {
		t.Error(`辞書外のローマ字が誤って収録されている`)
	}
}

// TestComposeKana は半角カナ折り畳み由来の「基底の仮名 + 結合濁点/半濁点」を
// 合成済みの 1 文字へ変換することを確認する（internal/normalize.Line が
// 1 ルーン = 1 ルーンを保つため未合成のまま返す結合文字を、辞書照合前に
// ここで合成する）。
func TestComposeKana(t *testing.T) {
	tests := []struct{ in, want string }{
		{"ダ", "ダ"}, // カタカナ濁点
		{"パ", "パ"}, // カタカナ半濁点
		{"が", "が"}, // ひらがな濁点
		{"ぱ", "ぱ"}, // ひらがな半濁点
		{"ヤマダ タロウ", "ヤマダ タロウ"}, // 文中の結合文字
		{"サトウ", "サトウ"},          // 結合文字を含まない場合はそのまま
		{"", ""},
		// 結合先が無い場合（対応する濁点/半濁点ペアが存在しない仮名）はそのまま残す。
		{"ア゙", "ア゙"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := ComposeKana(tt.in); got != tt.want {
				t.Errorf("ComposeKana(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestComposeKanaNoAllocWithoutCombiningMarks は結合文字を含まない入力に
// 対して割り当てなしで返すこと（正規化ホットパスに影響しないこと）を確認する。
func TestComposeKanaNoAllocWithoutCombiningMarks(t *testing.T) {
	in := "サトウサクラフリガナ住所電話番号"
	if n := testing.AllocsPerRun(10, func() { ComposeKana(in) }); n != 0 {
		t.Errorf("ComposeKana without combining marks allocated %v times, want 0", n)
	}
}

// TestIsPersonNameComposesHalfwidthOriginKana は、半角カナがフォールド後に
// 未合成の結合文字（U+3099/U+309A）を含む場合でも IsPersonName が辞書照合できる
// ことを確認する（normalize.Line("ﾔﾏﾀﾞ") → "ヤマダ"（合成前）相当の入力）。
func TestIsPersonNameComposesHalfwidthOriginKana(t *testing.T) {
	if !IsPersonName("ヤマダ タロウ") {
		t.Error(`IsPersonName("ヤマダ タロウ") = false, want true`)
	}
}
