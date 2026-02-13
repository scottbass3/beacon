package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
)

var writeClipboard = clipboard.WriteAll
var clipboardWriteAll = clipboard.WriteAll

func (m *Model) copySelectedTagReference() bool {
	ref, ok := m.selectedTagReferenceForCopy()
	if !ok {
		m.status = "No tag selected to copy"
		return false
	}
	if err := writeClipboard(ref); err != nil {
		m.status = fmt.Sprintf("Failed to copy %s: %v", ref, err)
		return false
	}
	m.status = fmt.Sprintf("Copied %s", ref)
	return true
}

func (m Model) selectedTagReferenceForCopy() (string, bool) {
	image, tag, ok := m.selectedTagImageAndTag()
	if !ok {
		return "", false
	}
	return formatTagReference(image, tag)
}

func (m Model) selectedTagImageAndTag() (string, string, bool) {
	list := m.listView()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(list.indices) {
		return "", "", false
	}
	index := list.indices[cursor]
	if index < 0 {
		return "", "", false
	}

	switch m.focus {
	case FocusTags:
		if !m.hasSelectedImage || index >= len(m.tags) {
			return "", "", false
		}
		return m.selectedImage.Name, m.tags[index].Name, true
	case FocusDockerHubTags:
		if index >= len(m.dockerHubTags) {
			return "", "", false
		}
		return m.dockerHubImage, m.dockerHubTags[index].Name, true
	case FocusGitHubTags:
		if index >= len(m.githubTags) {
			return "", "", false
		}
		return m.githubImage, m.githubTags[index].Name, true
	default:
		return "", "", false
	}
}

func formatTagReference(image, tag string) (string, bool) {
	image = strings.TrimSpace(image)
	tag = strings.TrimSpace(tag)
	if image == "" || tag == "" {
		return "", false
	}
	return image + ":" + tag, true
}
