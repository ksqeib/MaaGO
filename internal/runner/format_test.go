package runner

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatExtra(t *testing.T) {
	cases := []struct {
		what string
		raw  string
		want string
	}{
		{
			what: "ProductOfFacility",
			raw:  `{"facility":"Reception","index":0,"product":"General"}`,
			want: "会客室 #1 当前产品：综合",
		},
		{
			what: "EnterFacility",
			raw:  `{"facility":"Dorm","index":3}`,
			want: "进入设施：宿舍 #4",
		},
		{
			what: "RecruitTagsDetected",
			raw:  `{"tags":["近卫干员","狙击干员","医疗干员"]}`,
			want: "公招标签：近卫干员 / 狙击干员 / 医疗干员",
		},
		{
			what: "RecruitResult",
			raw:  `{"level":3,"result":[{"level":3,"opers":[{"id":"a","level":5,"name":"奥斯塔"},{"id":"b","level":5,"name":"慑砂"}],"tags":["狙击干员","群攻"]}]}`,
			want: "公招结果：3★ 狙击干员+群攻 → 奥斯塔、慑砂",
		},
		{
			what: "FightTimes",
			raw:  `{"sanity_cost":36,"series":6,"times_finished":1,"finished":true}`,
			want: "作战完成：已打 1 次，每次 36 理智，本次连战 6 次",
		},
		{
			what: "SanityBeforeStage",
			raw:  `{"current_sanity":40,"max_sanity":207}`,
			want: "当前理智：40 / 207",
		},
		{
			what: "ExceededLimit",
			raw:  `{"exec_times":0,"max_times":0,"task":"StartButton1"}`,
			want: "达到上限：开战次数（0 / 0）",
		},
		{
			what: "StageDrops",
			raw:  `{"drops":[{"dropType":"EXP_LMB_DROP","itemId":"4001","itemName":"龙门币","quantity":360},{"dropType":"FURNITURE","itemId":"furni","itemName":"furni","quantity":1},{"dropType":"NORMAL_DROP","itemId":"4006","itemName":"采购凭证","quantity":22}],"stage":{"stageCode":"AP-5"},"stars":3}`,
			want: "关卡 AP-5 掉落：龙门币×360、采购凭证×22（3★）",
		},
		{
			what: "UseMedicine",
			raw:  `{"count":1,"is_expiring":false,"medicines":[{"expire_days":-1,"inventory":2,"use":1}]}`,
			want: "使用理智药：1 瓶，库存 2",
		},
		{
			what: "SomeNewEvent",
			raw:  `{"foo":"bar","count":2}`,
			want: "SomeNewEvent：数量：2，foo：bar",
		},
	}
	for _, testCase := range cases {
		got := formatExtra(testCase.what, json.RawMessage(testCase.raw))
		if got != testCase.want {
			t.Errorf("%s: got %q, want %q", testCase.what, got, testCase.want)
		}
	}
	if got := formatExtra("Whatever", json.RawMessage(`not-json`)); got != "" {
		t.Errorf("invalid json should return empty, got %q", got)
	}
}

func TestFormatConnection(t *testing.T) {
	cases := []struct {
		what string
		raw  string
		why  string
		want string
	}{
		{"UuidGot", `{"uuid":"d2bf72d8f88aea36"}`, "", "设备 UUID：d2bf72d8f88aea36"},
		{"ResolutionGot", `{"width":1280,"height":720}`, "", "设备分辨率：1280×720"},
		{"Connected", `{"address":"192.168.10.160:5555","config":"General"}`, "", "设备已连接：192.168.10.160:5555（配置 General）"},
		{"ConnectFailed", `{}`, "timeout", "设备连接失败：timeout"},
		{"ScreencapCost", `{"avg":215,"max":238,"min":185,"fault_times":1}`, "", "截图耗时：平均 215ms，最大 238ms，最小 185ms，失败 1 次"},
		{"FastestWayToScreencap", `{"method":"RawWithGzip","cost":233}`, "", "截图方式：RawWithGzip（233ms）"},
	}
	for _, testCase := range cases {
		got := formatConnection(testCase.what, json.RawMessage(testCase.raw), testCase.why)
		if got != testCase.want {
			t.Errorf("%s: got %q, want %q", testCase.what, got, testCase.want)
		}
	}
	if got := formatExtra("SomeNewEvent", json.RawMessage(`{"nested":{"a":1}}`)); !strings.Contains(got, "a：1") {
		t.Errorf("nested json not rendered: %q", got)
	}
}
