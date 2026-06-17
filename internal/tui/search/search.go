package search

import (
	"strings"

	"github.com/mwdomino/cascade/internal/model"
	"github.com/sahilm/fuzzy"
)

// LocalFilter returns items whose title or any tag contains q (case-insensitive substring).
func LocalFilter(items []*model.Node, q string) []*model.Node {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return items
	}
	out := make([]*model.Node, 0, len(items))
	for _, it := range items {
		if matchesLocal(it, q) {
			out = append(out, it)
		}
	}
	return out
}

func matchesLocal(n *model.Node, q string) bool {
	if strings.Contains(strings.ToLower(n.Title()), q) {
		return true
	}
	for _, t := range n.FM.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

// GlobalFuzzy returns all nodes whose title + tags + body fuzzy-match q.
func GlobalFuzzy(all []*model.Node, q string) []*model.Node {
	if strings.TrimSpace(q) == "" {
		return all
	}
	corpus := make([]string, len(all))
	for i, n := range all {
		corpus[i] = n.Title() + " " + strings.Join(n.FM.Tags, " ") + " " + n.Body
	}
	matches := fuzzy.Find(q, corpus)
	out := make([]*model.Node, 0, len(matches))
	for _, m := range matches {
		out = append(out, all[m.Index])
	}
	return out
}
