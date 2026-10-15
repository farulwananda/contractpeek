package shape

import "strings"

// EscapeSegment encodes a path segment according to RFC 6901 JSON Pointer escaping,
// plus '~2' for a literal '*' character to preserve array item token distinction.
//
//	~  -> ~0
//	/  -> ~1
//	*  -> ~2
func EscapeSegment(seg string) string {
	res := strings.ReplaceAll(seg, "~", "~0")
	res = strings.ReplaceAll(res, "/", "~1")
	res = strings.ReplaceAll(res, "*", "~2")
	return res
}

// UnescapeSegment decodes an encoded path segment.
func UnescapeSegment(seg string) string {
	res := strings.ReplaceAll(seg, "~2", "*")
	res = strings.ReplaceAll(res, "~1", "/")
	res = strings.ReplaceAll(res, "~0", "~")
	return res
}

// JoinPath constructs a canonical pointer path from segments.
func JoinPath(parentPath, childSegment string) string {
	if parentPath == "" || parentPath == "#" {
		return "#/" + EscapeSegment(childSegment)
	}
	return parentPath + "/" + EscapeSegment(childSegment)
}

// JoinArrayItem constructs a canonical path pointing to an element inside an array.
func JoinArrayItem(arrayPath string) string {
	if arrayPath == "" || arrayPath == "#" {
		return "#/*"
	}
	return arrayPath + "/*"
}

// HasSubtreePrefix reports whether path is equal to or a child of subtreePrefix.
func HasSubtreePrefix(path, subtreePrefix string) bool {
	if path == subtreePrefix {
		return true
	}
	prefixSlash := subtreePrefix
	if !strings.HasSuffix(prefixSlash, "/") {
		prefixSlash += "/"
	}
	return strings.HasPrefix(path, prefixSlash)
}
