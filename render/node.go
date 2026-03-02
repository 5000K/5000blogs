package render

import (
	"fmt"
	"strings"
)

type NodeType string

type Node struct {
	Type        NodeType
	Literal     string            // Raw content
	Destination string            // URL for links
	Level       int               // For headers (1-6)
	Children    []*Node           // Nested content
	Attributes  map[string]string // HTML attributes
}

const (
	NodeDocument    NodeType = "document"
	NodeFrontmatter NodeType = "frontmatter"
	NodeParagraph   NodeType = "paragraph"
	NodeHeader      NodeType = "header"
	NodeBold        NodeType = "bold"
	NodeItalic      NodeType = "italic"
	NodeLink        NodeType = "link"
	NodeWikiLink    NodeType = "wikilink"
	NodeCode        NodeType = "code"
	NodeCodeBlock   NodeType = "codeblock"
	NodeText        NodeType = "text"
	NodeSoftBreak   NodeType = "softbreak"
)

// what can be nested inside what?
var allowedChildren = map[NodeType]map[NodeType]bool{
	NodeDocument: {
		NodeFrontmatter: true,
		NodeParagraph:   true,
		NodeHeader:      true,
		NodeCodeBlock:   true,
	},
	NodeParagraph: {
		NodeText:      true,
		NodeBold:      true,
		NodeItalic:    true,
		NodeLink:      true,
		NodeWikiLink:  true,
		NodeCode:      true,
		NodeSoftBreak: true,
	},
	NodeHeader: {
		NodeText:     true,
		NodeBold:     true,
		NodeItalic:   true,
		NodeLink:     true,
		NodeWikiLink: true,
		NodeCode:     true,
	},
	NodeBold: {
		NodeText:     true,
		NodeItalic:   true,
		NodeLink:     true,
		NodeWikiLink: true,
		NodeCode:     true,
	},
	NodeItalic: {
		NodeText:     true,
		NodeBold:     true,
		NodeLink:     true,
		NodeWikiLink: true,
		NodeCode:     true,
	},
	NodeLink: {
		NodeText:   true,
		NodeBold:   true,
		NodeItalic: true,
		// No nested links
	},
	// Leaf nodes
	NodeWikiLink:    {},
	NodeCodeBlock:   {},
	NodeCode:        {},
	NodeFrontmatter: {},
}

func (n *Node) CanContain(child NodeType) bool {
	if allowed, ok := allowedChildren[n.Type]; ok {
		return allowed[child]
	}
	return false
}

func (n *Node) AddChild(child *Node) error {
	if !n.CanContain(child.Type) {
		return fmt.Errorf("cannot add %s inside %s", child.Type, n.Type)
	}
	n.Children = append(n.Children, child)
	return nil
}

func (n *Node) String() string {
	var sb strings.Builder
	n.print(&sb, 0)
	return sb.String()
}

func (n *Node) print(sb *strings.Builder, indent int) {
	prefix := strings.Repeat("  ", indent)
	sb.WriteString(fmt.Sprintf("%s%s", prefix, n.Type))
	if n.Literal != "" {
		sb.WriteString(fmt.Sprintf(": %q", n.Literal))
	}
	if n.Destination != "" {
		sb.WriteString(fmt.Sprintf(" -> %q", n.Destination))
	}
	if n.Level > 0 {
		sb.WriteString(fmt.Sprintf(" [h%d]", n.Level))
	}
	sb.WriteString("\n")
	for _, child := range n.Children {
		child.print(sb, indent+1)
	}
}
