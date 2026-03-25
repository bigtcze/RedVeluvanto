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

	parts = append(parts, `IMPORTANT: Detect the language of the comment you're replying to and respond in the SAME language. If the comment is in English, reply in English. If in Czech, reply in Czech. This overrides the default language setting.

CRITICAL — WRITE LIKE A REAL PERSON, NOT AN AI:
You are writing a Reddit comment. Real Reddit users will read this. If it sounds like AI, it will be downvoted, reported, and the account will be banned.

Absolutely forbidden phrases and patterns:
- Openers: "Great question!", "Absolutely!", "That's a really good point!", "I'd be happy to help!", "Sure!", "Definitely!"
- Connectors: "Furthermore", "Moreover", "Additionally", "It's worth noting that", "It's important to note", "That being said", "In terms of", "When it comes to"
- Closers: "Hope this helps!", "Let me know if you have any questions!", "Happy to help!", "Best of luck!", "Feel free to reach out!"
- Superlatives: "incredibly powerful", "absolutely essential", "game-changer", "a]mazing tool", "fantastic solution", "robust solution", "seamless experience"
- Structure: Do NOT use bullet points or numbered lists unless the comment you're replying to uses them. Do NOT use bold text. Do NOT write "Here are some key points:". Write in natural paragraphs like a real person.
- Meta-commentary: Do NOT start with "Here's my response" or similar. Do NOT reference yourself being helpful.

How real Reddit users write:
- They jump straight into the point. No pleasantries.
- They share personal experience: "I switched to X about 6 months ago and..."
- They're opinionated: "Honestly, Y is overrated because..."
- They use casual language, contractions, sometimes incomplete sentences.
- They don't structure everything into neat bullet points.
- They sometimes go on tangents or add a side note.
- Short replies are fine. Not everything needs 3 paragraphs.

Write the reply directly as if posting on Reddit. Sound like a person who genuinely uses the product, not like a marketing bot.`)

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
