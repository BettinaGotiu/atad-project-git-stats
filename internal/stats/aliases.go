package stats

import (
	"atad-project/internal/gitlog"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Rule struct {
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Aliases []string `json:"aliases"`
}

type Aliases map[string]Rule

func LoadAliases(filename string) (Aliases, error) {
	if filename == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, err
	}
	return NewAliases(rules)
}

// NewAliases rejects ambiguous matches before any active rules are replaced.
func NewAliases(rules []Rule) (Aliases, error) {
	aliases := Aliases{}
	for _, rule := range rules {
		rule.Name = strings.TrimSpace(rule.Name)
		rule.Email = strings.TrimSpace(rule.Email)
		if rule.Name == "" || rule.Email == "" {
			return nil, errors.New("each alias rule needs a name and email")
		}
		for _, v := range append([]string{rule.Email}, rule.Aliases...) {
			key := strings.ToLower(strings.TrimSpace(v))
			if key == "" {
				return nil, errors.New("alias cannot be empty")
			}
			if prev, ok := aliases[key]; ok && (prev.Name != rule.Name || prev.Email != rule.Email) {
				return nil, fmt.Errorf("conflicting alias %q", v)
			}
			aliases[key] = rule
		}
	}
	return aliases, nil
}

func (a Aliases) Apply(c *gitlog.Commit) {
	for _, key := range []string{c.Email, c.Name} {
		if id, ok := a[strings.ToLower(strings.TrimSpace(key))]; ok {
			c.Name, c.Email = id.Name, id.Email
			return
		}
	}
}
