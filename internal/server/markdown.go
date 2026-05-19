package server

// StripFrontmatter removes YAML frontmatter delimited by ---\n...\n---\n from
// the beginning of src. If no valid frontmatter block is present, src is
// returned unchanged.
func StripFrontmatter(src []byte) []byte {
	return src
}

// RenderMarkdown renders Markdown src to HTML using goldmark with GFM
// extensions and chroma syntax highlighting. It calls StripFrontmatter on src
// before rendering.
func RenderMarkdown(src []byte) ([]byte, error) {
	return src, nil
}
