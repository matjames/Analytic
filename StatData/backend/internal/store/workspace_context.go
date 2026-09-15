package store

import "context"

type workspaceContextKey struct{}

// WithWorkspace binds the authenticated workspace to store operations.
func WithWorkspace(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, workspaceContextKey{}, workspaceID)
}

// WorkspaceID returns the workspace bound to a request, if any.
func WorkspaceID(ctx context.Context) string {
	if value, ok := ctx.Value(workspaceContextKey{}).(string); ok {
		return value
	}
	return ""
}

func workspaceVisible(ctx context.Context, resourceWorkspace string) bool {
	selected := WorkspaceID(ctx)
	return selected == "" || resourceWorkspace == "" || resourceWorkspace == selected
}

func bindWorkspace(ctx context.Context, resourceWorkspace *string) {
	if resourceWorkspace != nil && *resourceWorkspace == "" {
		*resourceWorkspace = WorkspaceID(ctx)
	}
}
