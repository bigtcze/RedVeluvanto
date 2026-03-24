package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
)

func RegisterKnowledgeRoutes(e *core.ServeEvent) {
	e.Router.POST("/api/knowledge/crawl", func(re *core.RequestEvent) error {
		var body struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}
		if body.URL == "" {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(body.URL)
		if err != nil {
			return re.JSON(http.StatusBadGateway, map[string]string{"error": "failed to fetch URL"})
		}
		defer resp.Body.Close()

		rawBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return re.JSON(http.StatusBadGateway, map[string]string{"error": "failed to read response"})
		}

		text := stripHTML(string(rawBytes))
		text = strings.TrimSpace(text)
		if len(text) > 50000 {
			text = text[:50000]
		}

		return re.JSON(http.StatusOK, map[string]string{
			"text": text,
			"url":  body.URL,
		})
	}).Bind(apis.RequireAuth())
}

func stripHTML(s string) string {
	s = reScript.ReplaceAllString(s, " ")
	s = reStyle.ReplaceAllString(s, " ")

	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			b.WriteByte(' ')
		case !inTag:
			b.WriteRune(r)
		}
	}

	out := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&apos;", "'",
		"&nbsp;", " ",
	).Replace(b.String())

	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
