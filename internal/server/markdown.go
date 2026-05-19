package server

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// StripFrontmatter removes YAML frontmatter delimited by ---\n...\n---\n from
// the beginning of src. If no valid frontmatter block is present, src is
// returned unchanged.
func StripFrontmatter(src []byte) []byte {
	if len(src) == 0 {
		return src
	}

	const delim = "---\n"
	if !bytes.HasPrefix(src, []byte(delim)) {
		return src
	}

	// Search for closing delimiter starting after the opening one.
	rest := src[len(delim):]
	idx := bytes.Index(rest, []byte(delim))
	if idx == -1 {
		return src
	}

	return rest[idx+len(delim):]
}

// RenderMarkdown renders Markdown src to HTML using goldmark with GFM
// extensions and chroma syntax highlighting. It calls StripFrontmatter on src
// before rendering.
func RenderMarkdown(src []byte) ([]byte, error) {
	stripped := StripFrontmatter(src)

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(stripped, &buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
