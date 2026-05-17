// SylastraClaws - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 SylastraClaws contributors

package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/conglinyizhi/SylastraClaws/pkg/logger"
	"github.com/conglinyizhi/SylastraClaws/pkg/media"
	"github.com/conglinyizhi/SylastraClaws/pkg/providers"
)

// resolveMediaRefs resolves media:// refs in messages.
// For every resolved file, it injects a structured file entry into the message
// content in the format:
//
//	- [file:///path] size mime uuid
//
// Non-media:// refs are passed through unchanged.
// Messages without media are returned as-is.
func resolveMediaRefs(messages []providers.Message, store media.MediaStore, _ int) []providers.Message {
	if store == nil {
		return messages
	}

	result := make([]providers.Message, 0, len(messages))

	for _, m := range messages {
		if len(m.Media) == 0 {
			result = append(result, m)
			continue
		}

		msg := m
		resolved := make([]string, 0, len(m.Media))
		var entries []string

		for _, ref := range m.Media {
			if !strings.HasPrefix(ref, "media://") {
				resolved = append(resolved, ref)
				continue
			}

			localPath, meta, err := store.ResolveWithMeta(ref)
			if err != nil {
				logger.WarnCF("agent", "Failed to resolve media ref", map[string]any{
					"ref":   ref,
					"error": err.Error(),
				})
				continue
			}

			info, err := os.Stat(localPath)
			if err != nil {
				logger.WarnCF("agent", "Failed to stat file", map[string]any{
					"path":  localPath,
					"error": err.Error(),
				})
				continue
			}

			mime := meta.ContentType
			if mime == "" {
				mime = "application/octet-stream"
			}

			fileID, err := uuid.NewV7()
			if err != nil {
				logger.WarnCF("agent", "Failed to generate uuid", map[string]any{
					"error": err.Error(),
				})
				continue
			}
			entry := formatFileEntry(localPath, info.Size(), mime, fileID.String())
			entries = append(entries, entry)
		}

		msg.Media = resolved
		if len(entries) > 0 {
			if msg.Content == "" {
				msg.Content = strings.Join(entries, "\n")
			} else {
				msg.Content = msg.Content + "\n" + strings.Join(entries, "\n")
			}
		}
		result = append(result, msg)
	}

	return result
}

// formatFileEntry creates a structured file entry in the format:
//
//	- [file:///path] size mime uuid
func formatFileEntry(localPath string, size int64, mime string, fileID string) string {
	return fmt.Sprintf("- [file://%s] %s %s %s", localPath, formatByteSize(size), mime, fileID)
}

// formatByteSize converts a byte count to a human-readable string.
func formatByteSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), suffixes[exp])
}

// buildArtifactTags resolves media refs to path tags for tool results.
func buildArtifactTags(store media.MediaStore, refs []string) []string {
	if store == nil || len(refs) == 0 {
		return nil
	}

	tags := make([]string, 0, len(refs))
	for _, ref := range refs {
		localPath, meta, err := store.ResolveWithMeta(ref)
		if err != nil {
			continue
		}
		mime := meta.ContentType
		if mime == "" {
			mime = "application/octet-stream"
		}
		fileID, err := uuid.NewV7()
		if err != nil {
			continue
		}
		tags = append(tags, formatFileEntry(localPath, fileSize(localPath), mime, fileID.String()))
	}
	return tags
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// buildProviderAttachments resolves media refs to provider attachments.
func buildProviderAttachments(store media.MediaStore, refs []string) []providers.Attachment {
	if store == nil || len(refs) == 0 {
		return nil
	}

	attachments := make([]providers.Attachment, 0, len(refs))
	for _, ref := range refs {
		attachment := providers.Attachment{Ref: ref}
		if _, meta, err := store.ResolveWithMeta(ref); err == nil {
			attachment.Filename = meta.Filename
			attachment.ContentType = meta.ContentType
		}
		attachments = append(attachments, attachment)
	}
	return attachments
}
