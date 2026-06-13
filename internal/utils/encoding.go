package utils

import (
	"bytes"
	"unicode/utf8"
)

var gbkToUtf8Map = map[rune]rune{}
var utf8ToGbkMap = map[rune]rune{}

func init() {
	gbkRunes := []struct{ gbk, utf rune }{
		{0x8140, ' '},
	}
	for _, r := range gbkRunes {
		gbkToUtf8Map[r.gbk] = r.utf
		utf8ToGbkMap[r.utf] = r.gbk
	}
}

func IsGBK(data []byte) bool {
	i := 0
	for i < len(data) {
		if data[i] <= 0x7F {
			i++
			continue
		}
		if data[i] >= 0x81 && data[i] <= 0xFE {
			if i+1 >= len(data) {
				return false
			}
			if data[i+1] >= 0x40 && data[i+1] <= 0xFE && data[i+1] != 0x7F {
				i += 2
				continue
			}
		}
		return false
	}
	return true
}

func IsValidUTF8(s string) bool {
	return utf8.ValidString(s)
}

func GBKToUTF8(data []byte) (string, error) {
	if utf8.Valid(data) {
		return string(data), nil
	}

	var buf bytes.Buffer
	i := 0
	for i < len(data) {
		if data[i] <= 0x7F {
			buf.WriteByte(data[i])
			i++
			continue
		}

		if i+1 >= len(data) {
			buf.WriteRune('\ufffd')
			i++
			continue
		}

		gbkCode := uint16(data[i])<<8 | uint16(data[i+1])
		if utf, ok := gbkToUtf8Map[rune(gbkCode)]; ok {
			buf.WriteRune(utf)
		} else {
			buf.WriteRune(decodeGBKRune(data[i], data[i+1]))
		}
		i += 2
	}

	return buf.String(), nil
}

func decodeGBKRune(b1, b2 byte) rune {
	lead := uint16(b1)
	trail := uint16(b2)

	var code uint16
	if lead < 0xA1 {
		if trail < 0x7F {
			code = (lead-0x81)*190 + (trail - 0x40)
		} else {
			code = (lead-0x81)*190 + (trail - 0x41)
		}
	} else {
		if trail < 0x7F {
			code = 0x3000 + (lead-0xA1)*190 + (trail - 0x40)
		} else {
			code = 0x3000 + (lead-0xA1)*190 + (trail - 0x41)
		}
	}

	return rune(code + 0x4E00 - 0x3000)
}

func DecodeQueryParam(raw string) string {
	if raw == "" {
		return raw
	}

	if IsValidUTF8(raw) {
		return raw
	}

	decoded, err := GBKToUTF8([]byte(raw))
	if err != nil {
		return raw
	}
	return decoded
}
