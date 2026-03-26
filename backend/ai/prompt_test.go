package ai

import (
	"fmt"
	"strings"
	"testing"
)

func emptyCall() string {
	return BuildPersonaPrompt(
		nil, nil, "", "", nil, "", nil, nil, nil, 0, "", "", "", nil,
	)
}

func TestBuildPersonaPrompt_EmptyInputs(t *testing.T) {
	result := emptyCall()

	if !strings.Contains(result, "You are a real Reddit user") {
		t.Error("empty call should contain base anti-AI-slop instructions")
	}
	if !strings.Contains(result, "Forbidden patterns") {
		t.Error("empty call should contain forbidden patterns section")
	}
	if !strings.Contains(result, "SUBREDDIT COMPLIANCE") {
		t.Error("empty call should contain subreddit compliance note")
	}
}

func TestBuildPersonaPrompt_SingleTrait(t *testing.T) {
	tests := []struct {
		name     string
		trait    string
		value    float64
		contains string
	}{
		{"formality low", "formality", 1, "Write very informally"},
		{"formality mid", "formality", 5, "balanced tone"},
		{"formality high", "formality", 9, "Write formally"},
		{"verbosity low", "verbosity", 0, "extremely brief"},
		{"verbosity mid", "verbosity", 4, "moderate length"},
		{"verbosity high", "verbosity", 10, "detailed and thorough"},
		{"humor low", "humor", 2, "completely serious"},
		{"humor mid", "humor", 6, "light humor"},
		{"humor high", "humor", 7, "sarcastic and witty"},
		{"empathy low", "empathy", 3, "purely factual"},
		{"empathy mid", "empathy", 4, "moderate understanding"},
		{"empathy high", "empathy", 8, "very empathetic"},
		{"confidence low", "confidence", 1, "hedging language"},
		{"confidence mid", "confidence", 5, "moderately confident"},
		{"confidence high", "confidence", 10, "very direct and confident"},
		{"expertise low", "expertise", 0, "curious learner"},
		{"expertise mid", "expertise", 6, "moderate knowledge"},
		{"expertise high", "expertise", 7, "deep expert"},
		{"controversy low", "controversy", 2, "Avoid any disagreement"},
		{"controversy mid", "controversy", 5, "gentle pushback"},
		{"controversy high", "controversy", 9, "challenge ideas directly"},
		{"emoji_usage low", "emoji_usage", 0, "Never use emoji"},
		{"emoji_usage mid", "emoji_usage", 4, "emoji sparingly"},
		{"emoji_usage high", "emoji_usage", 8, "emoji frequently"},
		{"typo_tolerance low", "typo_tolerance", 1, "perfect spelling"},
		{"typo_tolerance mid", "typo_tolerance", 5, "casual abbreviations"},
		{"typo_tolerance high", "typo_tolerance", 10, "intentional typos"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traits := map[string]float64{tt.trait: tt.value}
			result := BuildPersonaPrompt(traits, nil, "", "", nil, "", nil, nil, nil, 0, "", "", "", nil)

			if !strings.Contains(result, "## Writing Style") {
				t.Error("single trait should produce Writing Style section")
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("trait %s=%.0f should contain %q", tt.trait, tt.value, tt.contains)
			}
		})
	}
}

func TestBuildPersonaPrompt_TraitBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		contains string
	}{
		{"value 3 is low", 3, "Write very informally"},
		{"value 3.0 boundary", 3.0, "Write very informally"},
		{"value 4 is mid", 4, "balanced tone"},
		{"value 6 is mid", 6, "balanced tone"},
		{"value 6.0 boundary", 6.0, "balanced tone"},
		{"value 7 is high", 7, "Write formally"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traits := map[string]float64{"formality": tt.value}
			result := BuildPersonaPrompt(traits, nil, "", "", nil, "", nil, nil, nil, 0, "", "", "", nil)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formality=%.1f should contain %q, got:\n%s", tt.value, tt.contains, result)
			}
		})
	}
}

func TestBuildPersonaPrompt_ReplyGoals(t *testing.T) {
	tests := []struct {
		goal     string
		contains string
	}{
		{"help", "help. Don't mention any product"},
		{"promote", "natural opportunities to mention the product"},
		{"reputation", "building expert reputation"},
		{"traffic", "guide readers toward the website"},
		{"educate", "Share knowledge and educate"},
	}

	for _, tt := range tests {
		t.Run(tt.goal, func(t *testing.T) {
			result := BuildPersonaPrompt(nil, nil, tt.goal, "", nil, "", nil, nil, nil, 0, "", "", "", nil)

			if !strings.Contains(result, "## Goal") {
				t.Error("reply goal should produce Goal section")
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("goal %q should contain %q", tt.goal, tt.contains)
			}
		})
	}

	t.Run("unknown goal produces no Goal section", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "unknown", "", nil, "", nil, nil, nil, 0, "", "", "", nil)
		if strings.Contains(result, "## Goal") {
			t.Error("unknown goal should not produce Goal section")
		}
	})
}

func TestBuildPersonaPrompt_ReplyGoalWithDetail(t *testing.T) {
	result := BuildPersonaPrompt(nil, nil, "promote", "Focus on the free tier", nil, "", nil, nil, nil, 0, "", "", "", nil)
	if !strings.Contains(result, "Focus on the free tier") {
		t.Error("reply goal detail should appear in output")
	}
}

func TestBuildPersonaPrompt_BehaviorRules(t *testing.T) {
	rules := []string{"Never promise features", "Don't mention pricing"}
	result := BuildPersonaPrompt(nil, nil, "", "", rules, "", nil, nil, nil, 0, "", "", "", nil)

	if !strings.Contains(result, "## Behavior Rules") {
		t.Error("should contain Behavior Rules section")
	}
	for _, r := range rules {
		if !strings.Contains(result, r) {
			t.Errorf("should contain rule %q", r)
		}
	}

	t.Run("empty strings filtered", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", []string{"", ""}, "", nil, nil, nil, 0, "", "", "", nil)
		if strings.Contains(result, "## Behavior Rules") {
			t.Error("all-empty rules should not produce Behavior Rules section")
		}
	})
}

func TestBuildPersonaPrompt_CompetitorStance(t *testing.T) {
	tests := []struct {
		stance   string
		contains string
	}{
		{"ignore", "Never mention competitors"},
		{"acknowledge", "briefly acknowledge"},
		{"compare_fairly", "compare fairly"},
		{"differentiate", "what makes our approach different"},
	}

	for _, tt := range tests {
		t.Run(tt.stance, func(t *testing.T) {
			result := BuildPersonaPrompt(nil, nil, "", "", nil, tt.stance, nil, nil, nil, 0, "", "", "", nil)

			if !strings.Contains(result, "## Competitors") {
				t.Error("should contain Competitors section")
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("stance %q should contain %q", tt.stance, tt.contains)
			}
		})
	}

	t.Run("with competitor names", func(t *testing.T) {
		names := []string{"CompetitorA", "CompetitorB"}
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "acknowledge", names, nil, nil, 0, "", "", "", nil)
		if !strings.Contains(result, "CompetitorA, CompetitorB") {
			t.Error("competitor names should be listed comma-separated")
		}
		if !strings.Contains(result, "Competitors to be aware of") {
			t.Error("should contain competitor names label")
		}
	})

	t.Run("unknown stance produces no section", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "unknown", nil, nil, nil, 0, "", "", "", nil)
		if strings.Contains(result, "## Competitors") {
			t.Error("unknown stance should not produce Competitors section")
		}
	})
}

func TestBuildPersonaPrompt_ForbiddenWords(t *testing.T) {
	t.Run("local only", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, []string{"synergy", "leverage"}, nil, 0, "", "", "", nil)
		if !strings.Contains(result, "## Forbidden Words") {
			t.Error("should contain Forbidden Words section")
		}
		if !strings.Contains(result, "synergy") || !strings.Contains(result, "leverage") {
			t.Error("local forbidden words should appear")
		}
	})

	t.Run("global only", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, []string{"blockchain"}, 0, "", "", "", nil)
		if !strings.Contains(result, "blockchain") {
			t.Error("global forbidden words should appear")
		}
	})

	t.Run("merged local and global", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, []string{"local-word"}, []string{"global-word"}, 0, "", "", "", nil)
		if !strings.Contains(result, "local-word") || !strings.Contains(result, "global-word") {
			t.Error("both local and global forbidden words should appear")
		}
	})

	t.Run("empty strings filtered", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, []string{""}, []string{""}, 0, "", "", "", nil)
		if strings.Contains(result, "## Forbidden Words") {
			t.Error("all-empty forbidden words should not produce section")
		}
	})
}

func TestBuildPersonaPrompt_MaxLength(t *testing.T) {
	tests := []struct {
		name      string
		maxLength int
		contains  string
		absent    bool
	}{
		{"max length set", 500, "under 500 characters", false},
		{"zero means no limit", 0, "## Length", true},
		{"negative means no limit", -1, "## Length", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, nil, tt.maxLength, "", "", "", nil)
			if tt.absent {
				if strings.Contains(result, tt.contains) {
					t.Errorf("should NOT contain %q", tt.contains)
				}
			} else {
				if !strings.Contains(result, tt.contains) {
					t.Errorf("should contain %q", tt.contains)
				}
			}
		})
	}
}

func TestBuildPersonaPrompt_Language(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expect   bool
	}{
		{"cs produces language instruction", "cs", true},
		{"de produces language instruction", "de", true},
		{"en does NOT produce instruction", "en", false},
		{"empty does NOT produce instruction", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, nil, 0, tt.language, "", "", nil)
			hasSection := strings.Contains(result, "## Default Language")
			if tt.expect && !hasSection {
				t.Errorf("language %q should produce Default Language section", tt.language)
			}
			if !tt.expect && hasSection {
				t.Errorf("language %q should NOT produce Default Language section", tt.language)
			}
			if tt.expect && !strings.Contains(result, tt.language) {
				t.Errorf("should contain language code %q", tt.language)
			}
		})
	}
}

func TestBuildPersonaPrompt_Knowledge(t *testing.T) {
	t.Run("knowledge text only", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, nil, 0, "", "Our product does X", "", nil)
		if !strings.Contains(result, "## Product Knowledge") {
			t.Error("should contain Product Knowledge section")
		}
		if !strings.Contains(result, "Our product does X") {
			t.Error("knowledge text should appear")
		}
	})

	t.Run("knowledge cache only", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, nil, 0, "", "", "Cached info about product", nil)
		if !strings.Contains(result, "## Product Knowledge") {
			t.Error("should contain Product Knowledge section")
		}
		if !strings.Contains(result, "Cached info about product") {
			t.Error("knowledge cache should appear")
		}
	})

	t.Run("both combined", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, nil, 0, "", "text-part", "cache-part", nil)
		if !strings.Contains(result, "text-part") || !strings.Contains(result, "cache-part") {
			t.Error("both knowledge text and cache should appear")
		}
	})

	t.Run("empty knowledge no section", func(t *testing.T) {
		result := emptyCall()
		if strings.Contains(result, "## Product Knowledge") {
			t.Error("empty knowledge should not produce Product Knowledge section")
		}
	})
}

func TestBuildPersonaPrompt_Examples(t *testing.T) {
	examples := []string{"This is a great example reply.", "Another helpful response here."}
	result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, nil, 0, "", "", "", examples)

	if !strings.Contains(result, "## Example Responses") {
		t.Error("should contain Example Responses section")
	}
	if !strings.Contains(result, "Example 1:") {
		t.Error("first example should be numbered 1")
	}
	if !strings.Contains(result, "Example 2:") {
		t.Error("second example should be numbered 2")
	}
	for _, ex := range examples {
		if !strings.Contains(result, ex) {
			t.Errorf("should contain example text %q", ex)
		}
	}

	t.Run("empty strings filtered", func(t *testing.T) {
		result := BuildPersonaPrompt(nil, nil, "", "", nil, "", nil, nil, nil, 0, "", "", "", []string{"", ""})
		if strings.Contains(result, "## Example Responses") {
			t.Error("all-empty examples should not produce section")
		}
	})
}

func TestBuildPersonaPrompt_CustomTraits(t *testing.T) {
	customs := []string{"Always mention open-source", "Prefer short paragraphs"}
	result := BuildPersonaPrompt(nil, customs, "", "", nil, "", nil, nil, nil, 0, "", "", "", nil)

	if !strings.Contains(result, "## Additional Instructions") {
		t.Error("should contain Additional Instructions section")
	}
	for _, c := range customs {
		if !strings.Contains(result, c) {
			t.Errorf("should contain custom trait %q", c)
		}
	}
}

func TestBuildPersonaPrompt_FullCombination(t *testing.T) {
	result := BuildPersonaPrompt(
		map[string]float64{"formality": 8, "humor": 2},
		[]string{"Be concise"},
		"promote",
		"Mention free tier",
		[]string{"No pricing"},
		"differentiate",
		[]string{"Acme Corp"},
		[]string{"synergy"},
		[]string{"blockchain"},
		300,
		"cs",
		"We build doc mgmt",
		"",
		[]string{"Check out our tool"},
	)

	sections := []string{
		"## Writing Style",
		"## Additional Instructions",
		"## Goal",
		"## Behavior Rules",
		"## Competitors",
		"## Forbidden Words",
		"## Length",
		"## Default Language",
		"## Product Knowledge",
		"## Example Responses",
	}
	for _, s := range sections {
		if !strings.Contains(result, s) {
			t.Errorf("full combination should contain section %q", s)
		}
	}

	contents := []string{
		"Write formally",
		"completely serious",
		"Be concise",
		"natural opportunities",
		"Mention free tier",
		"No pricing",
		"what makes our approach different",
		"Acme Corp",
		"synergy",
		"blockchain",
		fmt.Sprintf("under %d characters", 300),
		"cs",
		"We build doc mgmt",
		"Check out our tool",
	}
	for _, c := range contents {
		if !strings.Contains(result, c) {
			t.Errorf("full combination should contain %q", c)
		}
	}
}
