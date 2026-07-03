// Package security provides input validation and prompt injection protection
// for all user-supplied content that is passed to AI services.
package security

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Limits on user-supplied content lengths (in Unicode characters).
const (
	MaxResumeContentLen  = 50_000 // ~15 pages of dense text
	MaxJobDescriptionLen = 10_000
	MaxShortFieldLen     = 300 // job title, company name, location, URL, role
)

// ErrInjectionDetected is returned when prompt injection patterns are found.
var ErrInjectionDetected = errors.New("content contains disallowed instructions")

// ErrInputTooLong is returned when a field exceeds the allowed character limit.
var ErrInputTooLong = errors.New("input exceeds maximum allowed length")

// injectionPatterns lists high-confidence prompt injection signatures.
// These combine instruction verbs with AI-context keywords and are unlikely
// to appear legitimately in a resume or job description.
var injectionPatterns = []*regexp.Regexp{
	// Classic "ignore instructions" attacks
	regexp.MustCompile(`(?i)\bignore\s+(all\s+)?(previous|prior|above|the\s+previous|these)\s+(instructions?|rules?|context|prompt|guidelines?|constraints?)\b`),
	// "Forget everything / disregard"
	regexp.MustCompile(`(?i)\b(forget|disregard|override)\s+(all|everything|previous|above|prior|the|these)\b`),
	// "You are now / act as / pretend to be"
	regexp.MustCompile(`(?i)\b(you\s+are\s+now|act\s+as\s+(a|an)|pretend\s+(you\s+are|you're|to\s+be))\b`),
	// Prompt/system leakage requests
	regexp.MustCompile(`(?i)\b(reveal|output|print|show|display|expose|leak|dump|repeat)\s+(your|the|my)?\s*(system\s+prompt|instructions?|api\s+key|secret|config|internal|prompt)\b`),
	// Model-specific injection tokens
	regexp.MustCompile(`(?i)(\[INST\]|<\|im_start\|>|<\|im_end\|>|<<SYS>>|<system>|</?(s\s*>|system\s*>)|\[\/INST\])`),
	// Role-switch patterns
	regexp.MustCompile(`(?i)\bnew\s+(role|instructions?|system|task|objective|directive)\s*:`),
	regexp.MustCompile(`(?i)\byour\s+(new\s+)?(instructions?|role|task|objective|directive)\s+(is|are)\b`),
	// "Do anything now" / DAN-style jailbreaks
	regexp.MustCompile(`(?i)\b(jailbreak|DAN|do\s+anything\s+now|developer\s+mode)\b`),
	// Override / bypass safety
	regexp.MustCompile(`(?i)\b(bypass|override|disable|remove)\s+(your\s+)?(safety|filter|restriction|limit|guideline|rule|constraint)\b`),
	// Instruction injections disguised as data
	regexp.MustCompile(`(?i)\bSTOP[.\s]+Ignore\s+the\s+(above|previous|prior)\b`),
	// Prompt chaining — trying to append a second task
	regexp.MustCompile(`(?i)\bInstead[,\s]+(respond|output|return|generate|write|produce|ignore)\b`),
}

// ValidateResumeContent checks resume text for injection patterns and size limits.
func ValidateResumeContent(content string) error {
	if err := checkLength(content, MaxResumeContentLen, "resume content"); err != nil {
		return err
	}
	return detectInjection(content)
}

// ValidateJobDescription checks a job description for injection patterns and size.
func ValidateJobDescription(desc string) error {
	if err := checkLength(desc, MaxJobDescriptionLen, "job description"); err != nil {
		return err
	}
	return detectInjection(desc)
}

// ValidateShortField checks short user-supplied fields (title, company, location, URL).
func ValidateShortField(value, fieldName string) error {
	if err := checkLength(value, MaxShortFieldLen, fieldName); err != nil {
		return err
	}
	return detectInjection(value)
}

// SanitizeForPrompt strips null bytes and control characters from a string
// without altering legitimate content. It does NOT truncate.
func SanitizeForPrompt(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		// Keep printable characters, spaces, tabs, and newlines.
		if r == '\t' || r == '\n' || r == '\r' || (r >= 0x20 && r != 0x7F) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// checkLength returns ErrInputTooLong when the Unicode character count exceeds max.
func checkLength(s string, max int, fieldName string) error {
	n := utf8.RuneCountInString(s)
	if n > max {
		return fmt.Errorf("%w: %s (%d chars, max %d)", ErrInputTooLong, fieldName, n, max)
	}
	return nil
}

// detectInjection scans s against all injection patterns.
func detectInjection(s string) error {
	for _, re := range injectionPatterns {
		if re.MatchString(s) {
			return fmt.Errorf("%w: suspicious instruction pattern detected", ErrInjectionDetected)
		}
	}
	return nil
}
