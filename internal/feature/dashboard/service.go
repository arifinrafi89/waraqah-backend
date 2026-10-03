// Package dashboard is the staff dashboard (port of dashboard_fake_store.dart) and the search log
// behind its top searches (search_log.dart). Every number comes from its own feature through a
// small read interface; the dashboard never reads other features' tables.
package dashboard

import (
	"context"
	"log/slog"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/feature/bookrequest"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/catalogadmin"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/moderation"
	"github.com/arifinrafi89/waraqah-backend/internal/feature/orders"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/clock"
)

// The read interfaces, one per feature.
type (
	Orders interface {
		DashboardNumbers(ctx context.Context, from, to time.Time) (orders.Numbers, error)
	}
	Listings interface {
		ListingsInReview(ctx context.Context) (int, error)
	}
	Reports interface {
		OpenReports(ctx context.Context) ([]moderation.Report, error)
	}
	Disputes interface {
		OpenDisputes(ctx context.Context) (int, error)
	}
	SellBacks interface {
		QueueLength(ctx context.Context) (int, error)
	}
	Stock interface {
		LowStock(ctx context.Context) ([]catalogadmin.LowStockItem, error)
	}
	Requests interface {
		Demand(ctx context.Context) ([]bookrequest.Demand, error)
	}
)

// Dashboard is AdminDashboardModel, plus what else waits for staff (additive fields the app
// ignores today: returns, Sell Back books to grade, low stock).
type Dashboard struct {
	OrdersToday     int                  `json:"ordersToday"`
	SalesTodayBdt   int                  `json:"salesTodayBdt"`
	OrdersToShip    int                  `json:"ordersToShip"`
	ListingsWaiting int                  `json:"listingsWaiting"`
	OpenReports     int                  `json:"openReports"`
	OpenDisputes    int                  `json:"openDisputes"`
	TopSearches     []Search             `json:"topSearches"`
	TopRequested    []bookrequest.Demand `json:"topRequested"`
	ReturnsWaiting  int                  `json:"returnsWaiting"`
	SellBackWaiting int                  `json:"sellBackWaiting"`
	LowStock        int                  `json:"lowStock"`
}

// Service builds the dashboard.
type Service struct {
	Orders    Orders
	Listings  Listings
	Reports   Reports
	Disputes  Disputes
	SellBacks SellBacks
	Stock     Stock
	Requests  Requests
	Searches  *SearchLog
	Clock     clock.Clock
	Loc       *time.Location
	Log       *slog.Logger
}

// Get is DashboardFakeStore.json: today's numbers in Dhaka time, what is waiting, the top five
// searches and the five most requested books.
func (s *Service) Get(ctx context.Context) (*Dashboard, error) {
	now := s.Clock.Now().In(s.Loc)
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.Loc)
	n, err := s.Orders.DashboardNumbers(ctx, from, from.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	out := &Dashboard{OrdersToday: n.OrdersToday, SalesTodayBdt: n.SalesTodayBdt, OrdersToShip: n.ToShip, ReturnsWaiting: n.ReturnsWaiting}
	if out.ListingsWaiting, err = s.Listings.ListingsInReview(ctx); err != nil {
		return nil, err
	}
	reports, err := s.Reports.OpenReports(ctx)
	if err != nil {
		return nil, err
	}
	out.OpenReports = len(reports)
	if out.OpenDisputes, err = s.Disputes.OpenDisputes(ctx); err != nil {
		return nil, err
	}
	if out.SellBackWaiting, err = s.SellBacks.QueueLength(ctx); err != nil {
		return nil, err
	}
	low, err := s.Stock.LowStock(ctx)
	if err != nil {
		return nil, err
	}
	out.LowStock = len(low)
	if out.TopSearches, err = s.Searches.Top(ctx, 5); err != nil {
		return nil, err
	}
	demand, err := s.Requests.Demand(ctx)
	if err != nil {
		return nil, err
	}
	out.TopRequested = append([]bookrequest.Demand{}, demand[:min(5, len(demand))]...)
	return out, nil
}
