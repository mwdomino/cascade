package store

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/mwdomino/cascade/internal/model"
	"gopkg.in/yaml.v3"
)

func ReadIndex(path string) (model.Frontmatter, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return model.Frontmatter{}, "", fmt.Errorf("read %s: %w", path, err)
	}
	// adrg/frontmatter decodes into a map so we can preserve unknown keys.
	var m map[string]any
	body, err := frontmatter.Parse(bytes.NewReader(raw), &m)
	if err != nil {
		return model.Frontmatter{}, "", fmt.Errorf("parse frontmatter %s: %w", path, err)
	}
	fm := model.Frontmatter{Extra: map[string]any{}}
	for k, v := range m {
		switch k {
		case "title":
			fm.Title, _ = v.(string)
		case "status":
			if s, ok := v.(string); ok {
				fm.Status = model.Status(s)
			}
		case "type":
			if s, ok := v.(string); ok {
				fm.Type = model.NodeType(s)
			}
		case "created":
			fm.Created = coerceTime(v)
		case "updated":
			fm.Updated = coerceTime(v)
		case "tags":
			fm.Tags = coerceStringSlice(v)
		default:
			fm.Extra[k] = v
		}
	}
	return fm, string(body), nil
}

func WriteIndex(path string, fm model.Frontmatter, body string) error {
	now := time.Now().UTC()
	if fm.Created.IsZero() {
		fm.Created = now
	}
	fm.Updated = now

	out := map[string]any{}
	if fm.Title != "" {
		out["title"] = fm.Title
	}
	if fm.Status != "" {
		out["status"] = string(fm.Status)
	}
	if fm.Type != "" {
		out["type"] = string(fm.Type)
	}
	out["created"] = fm.Created.Format(time.RFC3339)
	out["updated"] = fm.Updated.Format(time.RFC3339)
	if len(fm.Tags) > 0 {
		out["tags"] = fm.Tags
	}
	for k, v := range fm.Extra {
		out[k] = v
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		return err
	}
	enc.Close()
	buf.WriteString("---\n")
	buf.WriteString(body)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func coerceTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func coerceStringSlice(v any) []string {
	if arr, ok := v.([]any); ok {
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
