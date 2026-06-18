package detect

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/baneido/jp-pii-detector/internal/piifixtures"
)

// Finding は出力スキーマではなく、生の PII を持つ Match は json:"-" で
// シリアライズ対象から外している。誤って Finding を直接 marshal しても
// 生値が漏れないことを固定する回帰テスト（正規の出力は internal/report の
// jsonFinding を経由し、値はマスクされる）。
func TestFindingMarshalDoesNotLeakRawMatch(t *testing.T) {
	piifixtures.Require(t)
	raw := piifixtures.MustGet(t, "detect.finding_phone")
	f := Finding{RuleID: "jp-phone-number", File: "f.txt", Line: 1, Column: 1, Match: raw}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), raw) {
		t.Fatalf("marshaled Finding leaked raw match %q: %s", raw, b)
	}
}
