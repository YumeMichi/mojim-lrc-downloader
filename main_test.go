package main

import "testing"

func TestStripLrc(t *testing.T) {
	input := "[00:01.00]第一句\n[00:02.00]作詞: someone\n[00:03.00]第二句\n无时间戳"
	got := stripLrc(input)
	want := "第一句\n第二句"
	if got != want {
		t.Fatalf("stripLrc() = %q, want %q", got, want)
	}
}

func TestParseLrc(t *testing.T) {
	input := "[00:00.10]开场\n[00:01.00]第一句\n[00:02.50]第二句\n[00:03.00]作曲: someone"
	got := parseLrc(input)
	if len(got) != 3 {
		t.Fatalf("len(parseLrc()) = %d, want 3", len(got))
	}

	if got[0].Start != 0 || got[0].End != 800 || got[0].Text != "开场" {
		t.Fatalf("line0 = %+v", got[0])
	}
	if got[1].Start != 800 || got[1].End != 2300 || got[1].Text != "第一句" {
		t.Fatalf("line1 = %+v", got[1])
	}
	if got[2].Start != 2300 || got[2].End != 0 || got[2].Text != "第二句" {
		t.Fatalf("line2 = %+v", got[2])
	}
}

func TestSanitizeFileName(t *testing.T) {
	got := sanitizeFileName(`1 a/b:c*?"<>|`)
	want := "1 a／b：c＊？＂＜＞｜"
	if got != want {
		t.Fatalf("sanitizeFileName() = %q, want %q", got, want)
	}
}
