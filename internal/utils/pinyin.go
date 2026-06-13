package utils

import (
	"strings"

	"github.com/mozillazg/go-pinyin"
)

var pinyinArgs = pinyin.NewArgs()

func ToPinyin(text string) string {
	if !IsChinese(text) {
		return ""
	}
	result := pinyin.Pinyin(text, pinyinArgs)
	var builder strings.Builder
	for _, s := range result {
		if len(s) > 0 {
			builder.WriteString(s[0])
		}
	}
	return strings.ToLower(builder.String())
}

func IsChinese(text string) bool {
	for _, r := range text {
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
	}
	return false
}

func IsPinyin(text string) bool {
	if len(text) == 0 {
		return false
	}
	for _, r := range text {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ') {
			return false
		}
	}
	return true
}
