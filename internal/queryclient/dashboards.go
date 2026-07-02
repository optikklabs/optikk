package queryclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DashboardPage mirrors query/internal/modules/dashboards.DashboardPage.
type DashboardPage struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	IconColor   string    `json:"icon_color,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	IsFavorite  bool      `json:"is_favorite"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DashboardWidget mirrors a dashboard widget.
type DashboardWidget struct {
	ID            int64           `json:"id"`
	Title         string          `json:"title,omitempty"`
	PanelType     string          `json:"panel_type"`
	LayoutVariant string          `json:"layout_variant,omitempty"`
	Spec          json.RawMessage `json:"spec"`
	Layout        json.RawMessage `json:"layout"`
	Position      int             `json:"position"`
}

// CreatePageRequest mirrors the backend DTO.
type CreatePageRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	IconColor   string   `json:"icon_color,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsFavorite  bool     `json:"is_favorite"`
}

// DashboardExport is the full export shape (page + widgets).
type DashboardExport struct {
	Page    DashboardPage     `json:"page"`
	Widgets []DashboardWidget `json:"widgets"`
}

// ListDashboards returns all dashboard pages.
func (c *Client) ListDashboards(ctx context.Context, search, tag string, limit, offset int) ([]DashboardPage, error) {
	path := fmt.Sprintf("/v1/dashboard-pages?limit=%d&offset=%d", limit, offset)
	if search != "" {
		path += "&search=" + search
	}
	if tag != "" {
		path += "&tag=" + tag
	}
	var pages []DashboardPage
	if err := c.do(ctx, "GET", path, nil, &pages); err != nil {
		return nil, err
	}
	return pages, nil
}

// GetDashboard returns a single dashboard page.
func (c *Client) GetDashboard(ctx context.Context, id int64) (*DashboardPage, error) {
	var page DashboardPage
	if err := c.do(ctx, "GET", fmt.Sprintf("/v1/dashboard-pages/%d", id), nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// CreateDashboard creates a new dashboard page.
func (c *Client) CreateDashboard(ctx context.Context, req CreatePageRequest) (*DashboardPage, error) {
	var page DashboardPage
	if err := c.do(ctx, "POST", "/v1/dashboard-pages", req, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// UpdateDashboard updates an existing dashboard page.
func (c *Client) UpdateDashboard(ctx context.Context, id int64, req CreatePageRequest) (*DashboardPage, error) {
	var page DashboardPage
	if err := c.do(ctx, "PUT", fmt.Sprintf("/v1/dashboard-pages/%d", id), req, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// DeleteDashboard deletes a dashboard page.
func (c *Client) DeleteDashboard(ctx context.Context, id int64) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/v1/dashboard-pages/%d", id), nil, nil)
}

// ListWidgets returns widgets for a dashboard page.
func (c *Client) ListWidgets(ctx context.Context, pageID int64) ([]DashboardWidget, error) {
	var resp struct {
		Items []DashboardWidget `json:"items"`
	}
	if err := c.do(ctx, "GET", fmt.Sprintf("/v1/dashboard-pages/%d/dashboards", pageID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// ExportDashboard returns the full export shape (page + widgets).
func (c *Client) ExportDashboard(ctx context.Context, id int64) (*DashboardExport, error) {
	page, err := c.GetDashboard(ctx, id)
	if err != nil {
		return nil, err
	}
	widgets, err := c.ListWidgets(ctx, id)
	if err != nil {
		return nil, err
	}
	return &DashboardExport{Page: *page, Widgets: widgets}, nil
}

// ImportDashboard creates a dashboard from an export, optionally overriding the name.
func (c *Client) ImportDashboard(ctx context.Context, export DashboardExport, nameOverride string) (*DashboardPage, error) {
	req := CreatePageRequest{
		Name:        export.Page.Name,
		Description: export.Page.Description,
		Icon:        export.Page.Icon,
		IconColor:   export.Page.IconColor,
		Tags:        export.Page.Tags,
		IsFavorite:  export.Page.IsFavorite,
	}
	if nameOverride != "" {
		req.Name = nameOverride
	}
	page, err := c.CreateDashboard(ctx, req)
	if err != nil {
		return nil, err
	}
	// Re-create widgets on the new page.
	for _, w := range export.Widgets {
		body := map[string]any{
			"title":          w.Title,
			"panel_type":     w.PanelType,
			"layout_variant": w.LayoutVariant,
			"spec":           w.Spec,
			"layout":         w.Layout,
			"position":       w.Position,
		}
		if err := c.do(ctx, "POST", fmt.Sprintf("/v1/dashboard-pages/%d/dashboards", page.ID), body, nil); err != nil {
			return page, fmt.Errorf("created page %d but failed to import widget %q: %w", page.ID, w.Title, err)
		}
	}
	return page, nil
}
