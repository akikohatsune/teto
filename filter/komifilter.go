package filter

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type Decision struct {
	Blocked  bool
	Category string
	Reason   string
	Matches  []string
}

type PatternRule struct {
	Label   string
	Pattern *regexp.Regexp
}

type KomiFilter struct {
	Enabled                bool
	MaxCheckChars          int
	BlockResponseOnLeak    bool
	UserInjectionPatterns  []PatternRule
	UserPromptLeakPatterns []PatternRule
	ReplyLeakPatterns      []PatternRule
	ReplyStrongLeakMarkers []string
}

func NewKomiFilter(enabled bool, maxCheckChars int, blockResponseOnLeak bool) *KomiFilter {
	kf := &KomiFilter{
		Enabled:             enabled,
		MaxCheckChars:       maxCheckChars,
		BlockResponseOnLeak: blockResponseOnLeak,
	}

	kf.UserInjectionPatterns = []PatternRule{
		{"ignore_previous_instructions", regexp.MustCompile(`(?i)(?s)(?:ignore|disregard|forget|override|bypass|negate|overwrite|cancel|stop|drop).{0,100}(?:previous|prior|above|earlier|all|original|system|baseline|core).{0,100}(?:instructions?|rules?|system prompt|guardrails?|guidelines?|constraints?|directives?)`)},
		{"act_as_system_or_developer", regexp.MustCompile(`(?i)(?s)(?:act|behave|pretend|mimic|roleplay|simulate|assume).{0,60}(?:as|like|the role of|mode).{0,60}(?:system|developer|admin(?:istrator)?|root|god|kernel|super-user|technical support|creator)`)},
		{"disable_safety", regexp.MustCompile(`(?i)(?s)(?:disable|turn off|remove|skip|bypass|disable|suspend|deactivate|break).{0,60}(?:safety|policy|guardrails?|filters?|censorship|moderation|protections?|limits?)`)},
		{"role_spoofing_header", regexp.MustCompile(`(?i)(?m)^\s*(?:system|developer|assistant|user|admin|tool)\s*:`)},
		{"jailbreak_mode", regexp.MustCompile(`(?i)\b(?:jailbreak|dan mode|developer mode|aim mode|unfiltered mode|do anything now|broken constraints?|unshackled|free mode|based mode|sigma mode)\b`)},
		{"new_conversation_spoof", regexp.MustCompile(`(?i)(?:end of conversation|new conversation|start fresh|reboot|initialize|reset session|clear context)`)},
		{"output_formatting_override", regexp.MustCompile(`(?i)(?s)(?:output|respond|reply|format|print|return).{0,60}(?:only|exactly|using|in|as).{0,60}(?:json|code|raw|markdown|hex|base64|binary|python|terminal|shell|sql|csv)`)},
		{"obfuscation_payload", regexp.MustCompile(`(?i)(?:(?:\\x[0-9a-f]{2}){4,}|(?:\\u[0-9a-f]{4}){3,}|(?:&#x?[0-9a-f]+;){4,}|(?:[a-zA-Z0-9+/]{40,}={0,2}))`)},
		{"payload_injection_markers", regexp.MustCompile(`(?i)(?:<\|im_start\|>|<\|im_end\|>|\[INST\]|\[/INST\]|<<SYS>>|<\|system\|>|<\|user\|>|<\|assistant\|>)`)},
		{"forced_repetition", regexp.MustCompile(`(?i)(?s)\b(?:repeat|say|echo|print|write|copy)\b.{0,40}\b(?:after me|exactly|the following|these words|this word|this phrase|this exact phrase|my exact words)\b`)},
	}

	kf.UserPromptLeakPatterns = []PatternRule{
		{"request_system_prompt", regexp.MustCompile(`(?i)(?s)(?:show|reveal|print|dump|display|repeat|quote|return|expose|tell me|extract|what is|write out).{0,120}(?:system|developer|hidden|internal|original|initial|base|underlying|core).{0,120}(?:prompt|instructions?|message|rules?|personality|identity|logic|directives?|guidelines?)`)},
		{"request_markdown_source", regexp.MustCompile(`(?i)(?s)(?:show|reveal|print|dump|display|repeat|quote|return|expose|give me).{0,100}(?:markdown|source|file|text|raw content|codeblock)`)},
		{"rules_file_probe", regexp.MustCompile(`(?i)\b(?:system_rules\.md|rules source|rules markdown|system rules|gemini\.md|\.env|config\.py|config\.go|main\.go|bot\.go)\b`)},
	}

	kf.ReplyStrongLeakMarkers = []string{
		"you must follow these extra system rules loaded from markdown",
		"rules source:",
		"rules markdown:",
		"[call_profile_context]",
		"[message_content]",
		"[hidden_hook:Teto_fear]",
		"[attached_images=",
		"user calls Teto:",
		"Teto calls user:",
		"you are Teto, a playful ai assistant on discord",
		"default to english unless the user explicitly asks",
	}

	kf.ReplyLeakPatterns = []PatternRule{
		{"system_prompt_dump", regexp.MustCompile(`(?i)(?m)^\s*(?:system|developer|assistant)\s*(?:prompt|instructions?)\s*:`)},
		{"internal_prompt_phrase", regexp.MustCompile(`(?i)(?:internal|hidden|developer|baseline)\s+(?:prompt|instructions?|logic|rules?)`)},
	}

	return kf
}

func (kf *KomiFilter) InspectUserPrompt(text string) Decision {
	if !kf.Enabled {
		return Decision{Blocked: false}
	}

	sample := kf.prepareText(text)
	if sample == "" {
		return Decision{Blocked: false}
	}

	injectionHits := kf.collectMatches(sample, kf.UserInjectionPatterns)
	if len(injectionHits) > 0 {
		return Decision{
			Blocked:  true,
			Category: "prompt_injection",
			Reason:   "suspicious instruction override attempt",
			Matches:  injectionHits,
		}
	}

	leakHits := kf.collectMatches(sample, kf.UserPromptLeakPatterns)
	if len(leakHits) > 0 {
		return Decision{
			Blocked:  true,
			Category: "prompt_leak_request",
			Reason:   "suspicious system prompt discovery attempt",
			Matches:  leakHits,
		}
	}

	return Decision{Blocked: false}
}

func (kf *KomiFilter) InspectModelReply(text string) Decision {
	if !kf.Enabled || !kf.BlockResponseOnLeak {
		return Decision{Blocked: false}
	}

	sample := kf.prepareText(text)
	if sample == "" {
		return Decision{Blocked: false}
	}

	lowered := strings.ToLower(sample)
	var strongHits []string
	for _, marker := range kf.ReplyStrongLeakMarkers {
		if strings.Contains(lowered, marker) {
			strongHits = append(strongHits, marker)
		}
	}

	if len(strongHits) > 0 {
		return Decision{
			Blocked:  true,
			Category: "prompt_leak_response",
			Reason:   "model response exposed internal instruction markers",
			Matches:  strongHits,
		}
	}

	weakHits := kf.collectMatches(sample, kf.ReplyLeakPatterns)
	if len(weakHits) > 0 {
		return Decision{
			Blocked:  true,
			Category: "prompt_leak_response",
			Reason:   "model response resembles an internal prompt dump",
			Matches:  weakHits,
		}
	}

	return Decision{Blocked: false}
}

func (kf *KomiFilter) UserBlockMessage(decision Decision) string {
	if decision.Category == "prompt_injection" {
		return "komifilter! phát hiện nỗ lực thay đổi quy tắc hệ thống. vui lòng đặt câu hỏi trực tiếp mà không cố gắng bỏ qua các ràng buộc."
	}
	return "komifilter! phát hiện yêu cầu rò rỉ thông tin nội bộ. tôi không được phép tiết lộ quy tắc hoặc hướng dẫn hệ thống."
}

func (kf *KomiFilter) ReplyBlockMessage() string {
	return "komifilter! nội dung phản hồi bị chặn vì chứa thông tin nhạy cảm của hệ thống. vui lòng thử lại với một yêu cầu khác."
}

func (kf *KomiFilter) prepareText(text string) string {
	if text == "" {
		return ""
	}

	runesLimit := kf.MaxCheckChars
	if len(text) > runesLimit {
		text = text[:runesLimit]
	}

	// Unicode normalization (NFC)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, text)

	// Remove common obfuscation characters
	cleaned := regexp.MustCompile(`[\x{200b}-\x{200d}\x{feff}\x{00ad}]`).ReplaceAllString(normalized, "")

	return strings.TrimSpace(cleaned)
}

func (kf *KomiFilter) collectMatches(text string, rules []PatternRule) []string {
	var found []string
	for _, rule := range rules {
		if rule.Pattern.MatchString(text) {
			found = append(found, rule.Label)
		}
	}
	return found
}


