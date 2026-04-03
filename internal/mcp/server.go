package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/wesm/msgvault/internal/config"
	"github.com/wesm/msgvault/internal/embedding"
	"github.com/wesm/msgvault/internal/extractor"
	"github.com/wesm/msgvault/internal/pii"
	"github.com/wesm/msgvault/internal/query"
	"github.com/wesm/msgvault/internal/vector"
)

// Tool name constants.
const (
	ToolSearchMessages    = "search_messages"
	ToolGetMessage        = "get_message"
	ToolGetAttachment     = "get_attachment"
	ToolExportAttachment  = "export_attachment"
	ToolListMessages      = "list_messages"
	ToolGetStats          = "get_stats"
	ToolAggregate         = "aggregate"
	ToolStageDeletion     = "stage_deletion"
	ToolSearchAttachments = "search_attachments"
	ToolExtractAttachment = "extract_attachment"
)

// Common argument helpers for recurring tool option definitions.

func withLimit(defaultDesc string) mcp.ToolOption {
	return mcp.WithNumber("limit",
		mcp.Description("Maximum results to return (default "+defaultDesc+")"),
	)
}

func withOffset() mcp.ToolOption {
	return mcp.WithNumber("offset",
		mcp.Description("Number of results to skip for pagination (default 0)"),
	)
}

func withAfter() mcp.ToolOption {
	return mcp.WithString("after",
		mcp.Description("Only messages after this date (YYYY-MM-DD)"),
	)
}

func withBefore() mcp.ToolOption {
	return mcp.WithString("before",
		mcp.Description("Only messages before this date (YYYY-MM-DD)"),
	)
}

func withAccount() mcp.ToolOption {
	return mcp.WithString("account",
		mcp.Description("Filter by account email address (use get_stats to list available accounts)"),
	)
}

// Serve creates an MCP server with email archive tools and serves over stdio.
// It blocks until stdin is closed or the context is cancelled.
// dataDir is the base data directory (e.g., ~/.msgvault) used for deletions.
func Serve(ctx context.Context, engine query.Engine, attachmentsDir, dataDir string, cfg *config.Config) error {
	// Initialize PII filter
	piiFilter, err := pii.NewFilter()
	if err != nil {
		return fmt.Errorf("failed to initialize PII filter: %w", err)
	}

	s := server.NewMCPServer(
		"msgvault",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Initialize extractor service
	extractorSvc := &extractor.ExtractorService{}

	// Initialize embedding service and vector store if enabled in config
	var embeddingSvc embedding.Service
	var vectorSvc vector.VectorStore

	if cfg != nil && cfg.Embedding.Enabled {
		ollamaClient := embedding.NewOllamaClient(cfg.Embedding.OllamaURL)
		embeddingSvc = embedding.NewEmbeddingService(ollamaClient, cfg.Embedding.Model)

		if cfg.Vector.Store == "duckdb" {
			vectorDSN := "vector.duckdb"
			vectorSvc, err = vector.NewDuckDBStore(vectorDSN)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to initialize vector store: %v\n", err)
				vectorSvc = nil
			}
		}
	}

	h := &handlers{
		engine:         engine,
		attachmentsDir: attachmentsDir,
		dataDir:        dataDir,
		piiFilter:      piiFilter,
		extractor:      extractorSvc,
		embedding:      embeddingSvc,
		vectorStore:    vectorSvc,
	}

	s.AddTool(searchMessagesTool(), h.searchMessages)
	s.AddTool(getMessageTool(), h.getMessage)
	s.AddTool(getAttachmentTool(), h.getAttachment)
	s.AddTool(exportAttachmentTool(), h.exportAttachment)
	s.AddTool(listMessagesTool(), h.listMessages)
	s.AddTool(getStatsTool(), h.getStats)
	s.AddTool(aggregateTool(), h.aggregate)
	s.AddTool(stageDeletionTool(), h.stageDeletion)
	s.AddTool(searchAttachmentsTool(), h.searchAttachments)
	s.AddTool(extractAttachmentTool(), h.extractAttachment)

	stdio := server.NewStdioServer(s)

	if vectorSvc != nil {
		defer vectorSvc.Close()
	}

	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}

func searchMessagesTool() mcp.Tool {
	return mcp.NewTool(ToolSearchMessages,
		mcp.WithDescription("Search emails using Gmail-like query syntax. Supports from:, to:, subject:, label:, has:attachment, before:, after:, and free text."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Gmail-style search query (e.g. 'from:alice subject:meeting after:2024-01-01')"),
		),
		withAccount(),
		mcp.WithBoolean("include_attachments",
			mcp.Description("Also search attachment content using semantic vector search (requires vector store)"),
		),
		withLimit("20"),
		withOffset(),
	)
}

func getMessageTool() mcp.Tool {
	return mcp.NewTool(ToolGetMessage,
		mcp.WithDescription("Get full message details including body text, recipients, labels, and attachments by message ID."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Message ID"),
		),
	)
}

func getAttachmentTool() mcp.Tool {
	return mcp.NewTool(ToolGetAttachment,
		mcp.WithDescription("Get attachment content by attachment ID. Returns metadata as text and the file content as an embedded resource blob. Use get_message first to find attachment IDs."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithNumber("attachment_id",
			mcp.Required(),
			mcp.Description("Attachment ID (from get_message response)"),
		),
	)
}

func exportAttachmentTool() mcp.Tool {
	return mcp.NewTool(ToolExportAttachment,
		mcp.WithDescription("Save an attachment to the local filesystem. Use this for file types that cannot be displayed inline (e.g. PDFs, documents). Returns the saved file path."),
		mcp.WithNumber("attachment_id",
			mcp.Required(),
			mcp.Description("Attachment ID (from get_message response)"),
		),
		mcp.WithString("destination",
			mcp.Description("Directory to save the file to (default: ~/Downloads)"),
		),
	)
}

func listMessagesTool() mcp.Tool {
	return mcp.NewTool(ToolListMessages,
		mcp.WithDescription("List messages with optional filters. Returns message summaries sorted by date."),
		mcp.WithReadOnlyHintAnnotation(true),
		withAccount(),
		mcp.WithString("from",
			mcp.Description("Filter by sender email address"),
		),
		mcp.WithString("to",
			mcp.Description("Filter by recipient email address"),
		),
		mcp.WithString("label",
			mcp.Description("Filter by Gmail label"),
		),
		withAfter(),
		withBefore(),
		mcp.WithBoolean("has_attachment",
			mcp.Description("Only messages with attachments"),
		),
		withLimit("20"),
		withOffset(),
	)
}

func getStatsTool() mcp.Tool {
	return mcp.NewTool(ToolGetStats,
		mcp.WithDescription("Get archive overview: total messages, size, attachment count, and accounts."),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

func aggregateTool() mcp.Tool {
	return mcp.NewTool(ToolAggregate,
		mcp.WithDescription("Get grouped statistics (e.g. top senders, domains, labels, or message volume over time)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("group_by",
			mcp.Required(),
			mcp.Description("Dimension to group by"),
			mcp.Enum("sender", "recipient", "domain", "label", "time"),
		),
		withAccount(),
		withLimit("50"),
		withAfter(),
		withBefore(),
	)
}

func stageDeletionTool() mcp.Tool {
	return mcp.NewTool(ToolStageDeletion,
		mcp.WithDescription("Stage messages for deletion. Use EITHER 'query' (Gmail-style search) OR structured filters (from, domain, label, etc.), not both. Does NOT delete immediately - run 'msgvault delete-staged' CLI command to execute staged deletions."),
		withAccount(),
		mcp.WithString("query",
			mcp.Description("Gmail-style search query (e.g. 'from:linkedin subject:job alert'). Cannot be combined with structured filters."),
		),
		mcp.WithString("from",
			mcp.Description("Filter by sender email address"),
		),
		mcp.WithString("domain",
			mcp.Description("Filter by sender domain (e.g. 'linkedin.com')"),
		),
		mcp.WithString("label",
			mcp.Description("Filter by Gmail label (e.g. 'CATEGORY_PROMOTIONS')"),
		),
		withAfter(),
		withBefore(),
		mcp.WithBoolean("has_attachment",
			mcp.Description("Only messages with attachments"),
		),
	)
}

func searchAttachmentsTool() mcp.Tool {
	return mcp.NewTool(ToolSearchAttachments,
		mcp.WithDescription("Search attachment content using semantic vector search. Finds documents similar to your query based on meaning, not just keywords."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Semantic search query (e.g. 'find contracts about payment terms')"),
		),
		withLimit("10"),
		mcp.WithString("attachment_types",
			mcp.Description("Filter by attachment types: pdf, docx, txt (comma-separated)"),
		),
	)
}

func extractAttachmentTool() mcp.Tool {
	return mcp.NewTool(ToolExtractAttachment,
		mcp.WithDescription("Extract and index text content from a specific attachment for semantic search."),
		mcp.WithNumber("attachment_id",
			mcp.Required(),
			mcp.Description("Attachment ID to extract"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force re-extraction even if already extracted"),
		),
	)
}
