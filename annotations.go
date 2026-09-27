package main

import "github.com/mark3labs/mcp-go/mcp"

// Tool behaviour hints, as defined by the MCP specification. Clients use them
// to decide which calls need user confirmation, so every tool must have an
// entry here; TestEveryToolIsAnnotated enforces that.
//
// openWorldHint is true only for tools that make Miniflux reach out to the
// wider internet (fetching a feed or a web page, or pushing an entry to a
// third-party integration). Every other tool only touches this Miniflux
// instance.
var (
	annotateRead = mcp.ToolAnnotation{
		ReadOnlyHint:  mcp.ToBoolPtr(true),
		OpenWorldHint: mcp.ToBoolPtr(false),
	}
	annotateReadOpenWorld = mcp.ToolAnnotation{
		ReadOnlyHint:  mcp.ToBoolPtr(true),
		OpenWorldHint: mcp.ToBoolPtr(true),
	}
	// Adds something new; running it twice creates two of it.
	annotateCreate = mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}
	// Sets state to a given value; repeating the call changes nothing more,
	// and the previous value is easy to set back.
	annotateSet = mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}
	// Asks Miniflux to fetch feeds now; only adds entries.
	annotateRefresh = mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	}
	// Overwrites existing state in a way that is not trivially undone.
	annotateOverwrite = mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}
	annotateDelete = mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}
)

var toolAnnotations = map[string]mcp.ToolAnnotation{
	// Feeds
	"get_feeds":         annotateRead,
	"get_feed":          annotateRead,
	"create_feed":       withOpenWorld(annotateCreate),
	"update_feed":       annotateOverwrite,
	"delete_feed":       annotateDelete,
	"refresh_feed":      annotateRefresh,
	"refresh_all_feeds": annotateRefresh,
	"get_feed_entries":  annotateRead,
	"get_feed_entry":    annotateRead,
	"get_feed_icon":     annotateRead,
	"mark_feed_as_read": annotateOverwrite,

	// Entries
	"get_entries":            annotateRead,
	"get_entry":              annotateRead,
	"update_entry_status":    annotateSet,
	"toggle_starred":         withIdempotent(annotateSet, false),
	"save_entry":             withIdempotent(withOpenWorld(annotateSet), false),
	"fetch_original_content": annotateReadOpenWorld,
	"mark_all_as_read":       annotateOverwrite,

	// Categories
	"get_categories":        annotateRead,
	"create_category":       annotateCreate,
	"update_category":       annotateOverwrite,
	"delete_category":       annotateDelete,
	"get_category_feeds":    annotateRead,
	"get_category_entries":  annotateRead,
	"get_category_entry":    annotateRead,
	"mark_category_as_read": annotateOverwrite,
	"refresh_category":      annotateRefresh,

	// Users
	"get_users":            annotateRead,
	"get_me":               annotateRead,
	"get_user_by_id":       annotateRead,
	"get_user_by_username": annotateRead,
	"create_user":          annotateCreate,
	"delete_user":          annotateDelete,

	// System
	"get_version":    annotateRead,
	"healthcheck":    annotateRead,
	"fetch_counters": annotateRead,
	"discover":       annotateReadOpenWorld,
	"export":         annotateRead,
	"flush_history":  annotateDelete,

	// API keys
	"get_api_keys":   annotateRead,
	"create_api_key": annotateCreate,
	"delete_api_key": annotateDelete,

	// Icons and media
	"get_icon":      annotateRead,
	"get_enclosure": annotateRead,
}

func withOpenWorld(a mcp.ToolAnnotation) mcp.ToolAnnotation {
	a.OpenWorldHint = mcp.ToBoolPtr(true)
	return a
}

func withIdempotent(a mcp.ToolAnnotation, idempotent bool) mcp.ToolAnnotation {
	a.IdempotentHint = mcp.ToBoolPtr(idempotent)
	return a
}
