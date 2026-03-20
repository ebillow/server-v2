package make_cfg

import (
	"strings"
	"unicode"
)

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return ""
	}

	// Capitalize subsequent words
	var camelCase string
	for i := 0; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			r := []rune(parts[i])
			r[0] = unicode.ToUpper(r[0])
			camelCase += string(r)
		}
	}
	return camelCase
}
