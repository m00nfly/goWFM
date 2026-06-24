package handlers

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildAttachmentDisposition 构造符合 RFC 5987 的 Content-Disposition 响应头。
// 同时提供 ASCII 的 filename 回退（供不支持 filename* 的旧客户端使用），
// 以及 UTF-8 百分号编码的 filename*（支持中文等非 ASCII 文件名）。
func BuildAttachmentDisposition(filename string) string {
	asciiName := toASCIIFallback(filename)
	if asciiName == "" {
		asciiName = "download"
	}
	// 仅在含非 ASCII 字符时才追加 filename*，保持对旧客户端的兼容
	if isASCII(filename) {
		return fmt.Sprintf(`attachment; filename="%s"`, asciiName)
	}
	encoded := url.PathEscape(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiName, encoded)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// toASCIIFallback 生成 ASCII 可安全放入 quoted-string 的回退文件名：
// 丢弃非 ASCII 字符，并剔除控制字符、引号、反斜杠等不安全字符。
func toASCIIFallback(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c > 127 || c < 0x20 || c == 0x7f {
			continue
		}
		if c == '"' || c == '\\' || c == '\r' || c == '\n' {
			continue
		}
		b.WriteByte(c)
	}
	return strings.TrimSpace(b.String())
}
