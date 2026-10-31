package gui

import (
	"testing"

	"github.com/yetone/magpie/internal/usage"
)

// #4375: the tray tooltip — magpie's name over today's total, with no cards
// (trayIdleTip), and the usage:today / usage:7d cards' own tooltip
// (trayTotalCard) — was hard-coded English while the window's UI and the
// tray menu translated. It reads in the tray's language, as the menu does
// (trayText), English where a language has no words for it.
func TestTrayUsageTipLang(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	totals := usage.Totals{Calls: 152, Input: 912_000, Output: 318_400, CacheRead: 4_900_000, CacheWrite: 200_000, Cost: 3.4, Unpriced: 2}

	// English stays as it was, letter for letter.
	if got, want := trayIdleTip(totals, "usd", 0, "en"), "magpie\nToday: 6.3M tokens (912K in · 318K out · 5.1M cached)\n≈$3.40+ · 152 requests"; got != want {
		t.Errorf("en idle tip:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayTotalCard("usage:today", totals, "usd", 0, "en").Name, "Today in magpie\n6.3M tokens (912K in · 318K out · 5.1M cached)\n≈$3.40+\n152 requests, 2 with no known price"; got != want {
		t.Errorf("en total tip:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayTotalCard("usage:7d", usage.Totals{Calls: 1, Input: 950}, "usd", 0, "en").Name, "Last 7 days in magpie\n950 tokens (950 in · 0 out)\n1 request"; got != want {
		t.Errorf("en one-request tip:\n%s\nwant\n%s", got, want)
	}

	// Simplified Chinese, as the Settings page's own words have it.
	if got, want := trayIdleTip(totals, "usd", 0, "zh"), "magpie\n今日：6.3M token（输入 912K · 输出 318K · 缓存 5.1M）\n≈$3.40+ · 152 个请求"; got != want {
		t.Errorf("zh idle tip:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayTotalCard("usage:today", totals, "usd", 0, "zh").Name, "今日经 magpie\n6.3M token（输入 912K · 输出 318K · 缓存 5.1M）\n≈$3.40+\n152 个请求，其中 2 个无已知价格"; got != want {
		t.Errorf("zh total tip:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayIdleTip(usage.Totals{}, "usd", 0, "zh"), "magpie\n今日：暂无请求"; got != want {
		t.Errorf("zh no requests:\n%s\nwant\n%s", got, want)
	}

	// Traditional Chinese, Japanese and German.
	if got, want := trayIdleTip(totals, "usd", 0, "zh-TW"), "magpie\n今日：6.3M token（輸入 912K · 輸出 318K · 快取 5.1M）\n≈$3.40+ · 152 個請求"; got != want {
		t.Errorf("zh-TW idle tip:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayIdleTip(usage.Totals{}, "usd", 0, "zh-TW"), "magpie\n今日：尚無請求"; got != want {
		t.Errorf("zh-TW no requests:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayIdleTip(totals, "usd", 0, "ja"), "magpie\n今日：6.3M トークン（入力 912K · 出力 318K · キャッシュ 5.1M）\n≈$3.40+ · 152 件のリクエスト"; got != want {
		t.Errorf("ja idle tip:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayIdleTip(usage.Totals{}, "usd", 0, "ja"), "magpie\n今日：まだリクエストがありません"; got != want {
		t.Errorf("ja no requests:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayIdleTip(totals, "usd", 0, "de"), "magpie\nHeute: 6.3M Token (912K in · 318K out · 5.1M gecached)\n≈$3.40+ · 152 Anfragen"; got != want {
		t.Errorf("de idle tip:\n%s\nwant\n%s", got, want)
	}
	if got, want := trayIdleTip(usage.Totals{}, "usd", 0, "de"), "magpie\nHeute: noch keine Anfragen"; got != want {
		t.Errorf("de no requests:\n%s\nwant\n%s", got, want)
	}
}
