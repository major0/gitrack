package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// NewUUIDv7 returns a time-ordered UUIDv7 (RFC 9562) in canonical hyphenated
// form. Time ordering means on-disk listings and shard directories sort
// chronologically for free (ADR-0002). It is minted offline with no
// coordination, so concurrent creation on different branches never collides.
func NewUUIDv7() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("reading randomness for uuid: %w", err)
	}

	// 48-bit big-endian Unix milliseconds in the first 6 bytes.
	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	// Version 7 in the high nibble of byte 6; variant 10 in the top bits of
	// byte 8. The remaining bits stay random.
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80

	return formatUUID(b), nil
}

// formatUUID renders 16 bytes as canonical 8-4-4-4-12 lowercase hex.
func formatUUID(b [16]byte) string {
	var sb strings.Builder
	sb.Grow(36)
	enc := hex.EncodeToString(b[:])
	sb.WriteString(enc[0:8])
	sb.WriteByte('-')
	sb.WriteString(enc[8:12])
	sb.WriteByte('-')
	sb.WriteString(enc[12:16])
	sb.WriteByte('-')
	sb.WriteString(enc[16:20])
	sb.WriteByte('-')
	sb.WriteString(enc[20:32])
	return sb.String()
}

// IsUUID reports whether s is a canonical hyphenated UUID (any version). It is a
// shape check, not a version check: the format accepts any UUID the tree holds.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !isHexDigit(r) {
				return false
			}
		}
	}
	return true
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}
