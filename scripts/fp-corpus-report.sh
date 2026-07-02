#!/usr/bin/env sh
# scripts/fp-corpus-report.sh
#
# 大規模公開 OSS コーパス 1 件分の偽陽性率（findings/MLoC）を集計する。
# 自リポジトリの dogfooding（既定で 0 件期待）だけでは陰性母数が小さすぎて
# 偽陽性率の実証にならないため、第三者に長期間精査されている著名 OSS を対象に
# .github/workflows/fp-corpus-report.yml から呼び出す想定（詳細は同ファイル参照）。
#
# 使い方:
#   fp-corpus-report.sh <corpus-name> <corpus-dir> <findings-json>
#
# <findings-json> は `jp-pii-detect scan --format json --exit-zero <corpus-dir>` の
# 出力（internal/report.JSON の {"findings": [...], "count": N} 形式。値は既定でマスク
# 済み）をファイルに保存したものを渡す。本スクリプトは rule_id と件数だけを集計し、
# マスク済みの match 値すら出力に含めない（生値はもちろん、マスク値も外部に残さない）。
#
# MLoC（Million Lines of Code）は <corpus-dir> 配下の全ファイルの物理行数
# （find <dir> -type f | xargs wc -l 相当）を 1,000,000 で割った値。専用の loc 計測
# ツールは使わず、単純な物理行数で十分とする（対応方針 (1) 参照）。
#
# 出力: 集計結果の JSON を標準出力へ。
set -eu

die() {
	printf '%s\n' "fp-corpus-report: $*" >&2
	exit 1
}

command -v jq >/dev/null 2>&1 || die "jq が見つかりません"

if [ "$#" -ne 3 ]; then
	die "usage: fp-corpus-report.sh <corpus-name> <corpus-dir> <findings-json>"
fi
corpus=$1
dir=$2
findings_json=$3

[ -n "$corpus" ] || die "corpus-name が空です"
[ -d "$dir" ] || die "ディレクトリが存在しません: $dir"
[ -f "$findings_json" ] || die "findings JSON が存在しません: $findings_json"

# 物理行数（.git は除外）。`find | xargs wc -l` はファイル数が 1 件か複数件かで
# "total" 行の有無が変わり集計を誤りやすいため、全ファイルを cat で連結してから
# 1 回だけ wc -l する（ファイル数によらず安全）。
lines=$(find "$dir" -type f -not -path '*/.git/*' -print0 | xargs -0 cat -- 2>/dev/null | wc -l | tr -d ' ')

jq -n \
	--arg corpus "$corpus" \
	--argjson lines "$lines" \
	--slurpfile report "$findings_json" \
	'
	($lines / 1000000.0) as $mloc
	| ($report[0].findings // []) as $findings
	| ($findings | length) as $total
	| {
		corpus: $corpus,
		physical_lines: $lines,
		mloc: $mloc,
		findings_total: $total,
		findings_per_mloc: (if $mloc > 0 then ($total / $mloc) else null end),
		by_rule: (
			$findings
			| group_by(.rule_id)
			| map({
				rule_id: .[0].rule_id,
				count: length,
				per_mloc: (if $mloc > 0 then (length / $mloc) else null end)
			})
			| sort_by(-.count)
		)
	}
	'
