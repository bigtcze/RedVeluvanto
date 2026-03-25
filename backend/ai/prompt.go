package ai

import (
	"fmt"
	"strings"
)

func BuildPersonaPrompt(
	traits map[string]float64,
	customTraits []string,
	replyGoal string,
	replyGoalDetail string,
	behaviorRules []string,
	competitorStance string,
	competitorNames []string,
	forbiddenWords []string,
	globalForbiddenWords []string,
	maxLength int,
	language string,
	knowledgeText string,
	knowledgeCache string,
	examples []string,
) string {
	var parts []string

	traitDescriptions := map[string]struct{ low, mid, high string }{
		"formality":      {"Write very informally, use slang and casual language", "Write in a balanced tone, neither too formal nor too casual", "Write formally, use proper grammar and professional language"},
		"verbosity":      {"Be extremely brief, 1-2 sentences max", "Write moderate length responses", "Be detailed and thorough, explain in depth"},
		"humor":          {"Be completely serious, no jokes", "Occasionally use light humor", "Be very sarcastic and witty, use humor frequently"},
		"empathy":        {"Be purely factual and objective", "Show moderate understanding", "Be very empathetic and understanding, acknowledge feelings"},
		"confidence":     {"Be cautious, use hedging language like 'maybe', 'perhaps'", "Be moderately confident", "Be very direct and confident, state things with certainty"},
		"expertise":      {"Write as a curious learner, ask questions", "Show moderate knowledge of the topic", "Write as a deep expert, use technical language when appropriate"},
		"controversy":    {"Avoid any disagreement, be agreeable", "Offer gentle pushback when appropriate", "Don't shy away from disagreement, challenge ideas directly"},
		"emoji_usage":    {"Never use emoji", "Use emoji sparingly, 1-2 max", "Use emoji frequently to express tone"},
		"typo_tolerance": {"Use perfect spelling and grammar", "Occasional casual abbreviations are ok", "Write casually with intentional typos and abbreviations"},
	}

	var traitLines []string
	for name, desc := range traitDescriptions {
		val, ok := traits[name]
		if !ok {
			continue
		}
		var instruction string
		switch {
		case val <= 3:
			instruction = desc.low
		case val <= 6:
			instruction = desc.mid
		default:
			instruction = desc.high
		}
		traitLines = append(traitLines, "- "+instruction)
	}
	if len(traitLines) > 0 {
		parts = append(parts, "## Writing Style\n"+strings.Join(traitLines, "\n"))
	}

	if len(customTraits) > 0 {
		var lines []string
		for _, t := range customTraits {
			if t != "" {
				lines = append(lines, "- "+t)
			}
		}
		if len(lines) > 0 {
			parts = append(parts, "## Additional Instructions\n"+strings.Join(lines, "\n"))
		}
	}

	goalInstructions := map[string]string{
		"help":       "Your primary goal is to help. Don't mention any product unless it directly solves the person's problem.",
		"promote":    "Look for natural opportunities to mention the product, but always provide genuine value first. Never force it.",
		"reputation": "Focus on building expert reputation. Mention the product only peripherally or not at all.",
		"traffic":    "Try to guide readers toward the website/link, but do it naturally by providing value first.",
		"educate":    "Share knowledge and educate. Use the product as an example when relevant but focus on teaching.",
	}
	if instr, ok := goalInstructions[replyGoal]; ok {
		goalSection := "## Goal\n" + instr
		if replyGoalDetail != "" {
			goalSection += "\n" + replyGoalDetail
		}
		parts = append(parts, goalSection)
	}

	if len(behaviorRules) > 0 {
		var lines []string
		for _, r := range behaviorRules {
			if r != "" {
				lines = append(lines, "- "+r)
			}
		}
		if len(lines) > 0 {
			parts = append(parts, "## Behavior Rules\n"+strings.Join(lines, "\n"))
		}
	}

	stanceInstructions := map[string]string{
		"ignore":         "Never mention competitors, even if asked directly.",
		"acknowledge":    "If competitors come up, briefly acknowledge they exist but don't elaborate.",
		"compare_fairly": "When competitors come up, compare fairly — mention pros and cons of both sides.",
		"differentiate":  "When competitors come up, emphasize what makes our approach different without directly criticizing them.",
	}
	if stance, ok := stanceInstructions[competitorStance]; ok {
		section := "## Competitors\n" + stance
		if len(competitorNames) > 0 {
			section += "\nCompetitors to be aware of: " + strings.Join(competitorNames, ", ")
		}
		parts = append(parts, section)
	}

	var allForbidden []string
	for _, w := range globalForbiddenWords {
		if w != "" {
			allForbidden = append(allForbidden, w)
		}
	}
	for _, w := range forbiddenWords {
		if w != "" {
			allForbidden = append(allForbidden, w)
		}
	}
	if len(allForbidden) > 0 {
		parts = append(parts, fmt.Sprintf("## Forbidden Words\nNever use these words or phrases: %s", strings.Join(allForbidden, ", ")))
	}

	if maxLength > 0 {
		parts = append(parts, fmt.Sprintf("## Length\nKeep your response under %d characters.", maxLength))
	}

	if language != "" && language != "en" {
		parts = append(parts, fmt.Sprintf("## Default Language\nDefault response language: %s", language))
	}

	kb := combineKnowledge(knowledgeText, knowledgeCache)
	if kb != "" {
		parts = append(parts, "## Product Knowledge\n"+kb)
	}

	if len(examples) > 0 {
		var lines []string
		for i, ex := range examples {
			if ex != "" {
				lines = append(lines, fmt.Sprintf("Example %d:\n%s", i+1, ex))
			}
		}
		if len(lines) > 0 {
			parts = append(parts, "## Example Responses\n"+strings.Join(lines, "\n\n"))
		}
	}

	parts = append(parts, `Detect the language of the target comment and reply in the SAME language. This overrides the default language setting.

You are a real Reddit user, not an AI assistant. Write like you're posting from your phone between meetings. If this sounds like AI, the account gets banned.

Forbidden patterns:
- Openers/closers: "Great question!", "Absolutely!", "Hope this helps!", "Let me know if you have questions!", "Happy to help!"
- Connectors: "Furthermore", "Moreover", "Additionally", "It's worth noting", "That being said"
- Superlatives: "game-changer", "robust solution", "incredibly powerful", "seamless experience"
- No bullet points or bold unless the parent comment uses them
- No meta-commentary ("Here's my response", "Here are some key points:")

Jump straight to your point. Share personal experience. Be opinionated. Short is fine.

SUBREDDIT COMPLIANCE: Subreddit rules ALWAYS override your persona goals. If rules prohibit self-promotion or ads, do NOT mention any product even if your goal is "promote" or "traffic". Focus on being genuinely helpful instead. If you had to suppress your goal due to rules, append on a new line: [⚠️ Subreddit rules restrict promotion — reply adjusted]`)

	return strings.Join(parts, "\n\n")
}

func combineKnowledge(knowledgeText, knowledgeCache string) string {
	var parts []string
	if knowledgeText != "" {
		parts = append(parts, knowledgeText)
	}
	if knowledgeCache != "" {
		parts = append(parts, knowledgeCache)
	}
	return strings.Join(parts, "\n\n")
}
