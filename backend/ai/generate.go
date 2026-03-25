package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"redveluvanto/reddit"
)

func (c *Client) GenerateReply(ctx context.Context, personaRecord *core.Record, threadRecord *core.Record, targetCommentID string) (string, error) {
	var traits map[string]float64
	unmarshalField(personaRecord.GetString("traits"), &traits)
	if traits == nil {
		traits = map[string]float64{}
	}

	var customTraits []string
	unmarshalField(personaRecord.GetString("custom_traits"), &customTraits)

	replyGoal := personaRecord.GetString("reply_goal")
	replyGoalDetail := personaRecord.GetString("reply_goal_detail")

	var behaviorRules []string
	unmarshalField(personaRecord.GetString("behavior_rules"), &behaviorRules)

	competitorStance := personaRecord.GetString("competitor_stance")

	var competitorNames []string
	unmarshalField(personaRecord.GetString("competitor_names"), &competitorNames)

	var forbiddenWords []string
	unmarshalField(personaRecord.GetString("forbidden_words"), &forbiddenWords)

	maxLength := personaRecord.GetInt("max_length")
	language := personaRecord.GetString("language")
	knowledgeText := personaRecord.GetString("knowledge_text")
	knowledgeCache := personaRecord.GetString("knowledge_cache")

	var examples []string
	unmarshalField(personaRecord.GetString("examples"), &examples)

	systemPrompt := BuildPersonaPrompt(
		traits, customTraits, replyGoal, replyGoalDetail,
		behaviorRules, competitorStance, competitorNames,
		forbiddenWords, maxLength, language,
		knowledgeText, knowledgeCache, examples,
	)

	subredditName := threadRecord.GetString("subreddit")
	title := threadRecord.GetString("title")
	body := threadRecord.GetString("body")
	author := threadRecord.GetString("author")
	score := threadRecord.GetInt("score")
	commentsJSON := threadRecord.GetString("comments_tree")
	rulesJSON := threadRecord.GetString("subreddit_rules")
	subredditDesc := threadRecord.GetString("subreddit_description")

	lookupID := targetCommentID
	if strings.HasPrefix(lookupID, "t1_") {
		lookupID = lookupID[3:]
	}

	rulesText := formatSubredditRules(rulesJSON)
	commentsText := formatCommentsTree(commentsJSON, lookupID)

	var targetDesc string
	if lookupID == "" {
		targetDesc = "the original post"
	} else {
		var comments []*reddit.Comment
		if err := json.Unmarshal([]byte(commentsJSON), &comments); err == nil {
			if target := findComment(comments, lookupID); target != nil {
				preview := target.Body
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				targetDesc = fmt.Sprintf("the comment by u/%s: %s", target.Author, preview)
			}
		}
		if targetDesc == "" {
			targetDesc = fmt.Sprintf("the comment with ID %s", lookupID)
		}
	}

	var userMsgParts []string
	if subredditName != "" {
		userMsgParts = append(userMsgParts, "=== SUBREDDIT ===\nr/"+subredditName)
	}
	if rulesText != "" {
		userMsgParts = append(userMsgParts, "=== SUBREDDIT RULES ===\n"+rulesText)
	}
	if subredditDesc != "" {
		userMsgParts = append(userMsgParts, "=== SUBREDDIT DESCRIPTION ===\n"+subredditDesc)
	}
	userMsgParts = append(userMsgParts, fmt.Sprintf("=== ORIGINAL POST ===\nTitle: %s\nAuthor: u/%s\nScore: %d\n\n%s", title, author, score, body))
	if commentsText != "" {
		userMsgParts = append(userMsgParts, "=== COMMENTS ===\n"+commentsText)
	}
	userMsgParts = append(userMsgParts, fmt.Sprintf("=== YOUR TARGET ===\nReply to %s\n\nGenerate your reply now.", targetDesc))

	complianceNote := `

CRITICAL RULE — SUBREDDIT COMPLIANCE:
Subreddit rules ALWAYS take priority over your persona goals. If the subreddit rules prohibit self-promotion, product links, advertising, or commercial content, you MUST comply even if your reply goal is "promote" or "traffic". In such cases, focus on being genuinely helpful without mentioning any product or link. When you had to suppress your persona goal due to subreddit rules, append this marker on a new line at the very end of your reply: [⚠️ Subreddit rules restrict promotion — reply adjusted]`

	messages := []Message{
		{Role: "system", Content: systemPrompt + complianceNote},
		{Role: "user", Content: strings.Join(userMsgParts, "\n\n")},
	}

	return c.ChatCompletion(ctx, messages, 0.7)
}

func unmarshalField(s string, v any) {
	if s == "" {
		return
	}
	_ = json.Unmarshal([]byte(s), v)
}

func formatSubredditRules(rulesJSON string) string {
	if rulesJSON == "" {
		return ""
	}
	var rules []struct {
		ShortName   string `json:"short_name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return ""
	}
	var lines []string
	for i, r := range rules {
		if r.ShortName != "" {
			entry := fmt.Sprintf("%d. %s", i+1, r.ShortName)
			if r.Description != "" {
				entry += ": " + r.Description
			}
			lines = append(lines, entry)
		}
	}
	return strings.Join(lines, "\n")
}

func formatCommentsTree(commentsJSON string, targetID string) string {
	if commentsJSON == "" {
		return ""
	}
	var comments []*reddit.Comment
	if err := json.Unmarshal([]byte(commentsJSON), &comments); err != nil {
		return ""
	}
	var sb strings.Builder
	writeComments(&sb, comments, 0, targetID)
	return sb.String()
}

func writeComments(sb *strings.Builder, comments []*reddit.Comment, depth int, targetID string) {
	for _, c := range comments {
		if c == nil {
			continue
		}
		var header string
		if depth == 0 {
			header = fmt.Sprintf("u/%s (score: %d):", c.Author, c.Score)
		} else {
			header = strings.Repeat("   ", depth-1) + "  └─ " + fmt.Sprintf("u/%s (score: %d):", c.Author, c.Score)
		}
		if c.ID == targetID {
			header += " [YOUR TARGET]"
		}
		sb.WriteString(header + "\n")
		contentPad := strings.Repeat("   ", depth) + "  "
		for _, line := range strings.Split(c.Body, "\n") {
			sb.WriteString(contentPad + line + "\n")
		}
		sb.WriteString("\n")
		if len(c.Replies) > 0 {
			writeComments(sb, c.Replies, depth+1, targetID)
		}
	}
}

func findComment(comments []*reddit.Comment, id string) *reddit.Comment {
	for _, c := range comments {
		if c == nil {
			continue
		}
		if c.ID == id {
			return c
		}
		if found := findComment(c.Replies, id); found != nil {
			return found
		}
	}
	return nil
}
