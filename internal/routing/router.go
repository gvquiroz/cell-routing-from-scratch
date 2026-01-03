package routing

import "fmt"

// ConfigProvider provides access to routing configuration
type ConfigProvider interface {
	GetRoutingTable() map[string]string
	GetCellEndpoints() map[string]string
	GetDefaultPlacement() string
}

// RouteReason indicates why a particular placement was chosen
type RouteReason string

const (
	ReasonDedicated RouteReason = "dedicated"
	ReasonTier      RouteReason = "tier"
	ReasonDefault   RouteReason = "default"
)

// RoutingDecision contains the result of a routing lookup
type RoutingDecision struct {
	PlacementKey string
	Reason       RouteReason
	EndpointURL  string
}

// Router handles routing decisions based on routing keys
type Router struct {
	configProvider ConfigProvider
}

// NewRouter creates a new Router with a config provider
func NewRouter(configProvider ConfigProvider) *Router {
	return &Router{
		configProvider: configProvider,
	}
}

// NewRouterWithMaps creates a new Router with static maps (for backward compatibility and tests)
func NewRouterWithMaps(
	routingTable map[string]string,
	cellEndpoints map[string]string,
	defaultPlacement string,
) *Router {
	return &Router{
		configProvider: &staticConfig{
			routingTable:     routingTable,
			cellEndpoints:    cellEndpoints,
			defaultPlacement: defaultPlacement,
		},
	}
}

// Route determines the placement and endpoint for a given routing key
func (r *Router) Route(routingKey string) (*RoutingDecision, error) {
	// Get current config atomically
	routingTable := r.configProvider.GetRoutingTable()
	cellEndpoints := r.configProvider.GetCellEndpoints()
	defaultPlacement := r.configProvider.GetDefaultPlacement()

	// Lookup placement (use default if not found or empty)
	placementKey, found := routingTable[routingKey]
	if !found || routingKey == "" {
		placementKey = defaultPlacement
	}

	// Determine reason
	reason := r.determineReason(routingKey, placementKey, found)

	// Lookup endpoint URL
	endpointURL, found := cellEndpoints[placementKey]
	if !found {
		return nil, fmt.Errorf("no endpoint configured for placement: %s", placementKey)
	}

	return &RoutingDecision{
		PlacementKey: placementKey,
		Reason:       reason,
		EndpointURL:  endpointURL,
	}, nil
}

// determineReason returns the routing reason based on the lookup result
func (r *Router) determineReason(routingKey, placementKey string, found bool) RouteReason {
	if !found || routingKey == "" {
		return ReasonDefault
	}

	if r.isTier(placementKey) {
		return ReasonTier
	}
	return ReasonDedicated
}

// isTier checks if the placement key is a shared tier
func (r *Router) isTier(placementKey string) bool {
	return placementKey == "tier1" || placementKey == "tier2" || placementKey == "tier3"
}

// staticConfig implements ConfigProvider for static/test configurations
type staticConfig struct {
	routingTable     map[string]string
	cellEndpoints    map[string]string
	defaultPlacement string
}

func (s *staticConfig) GetRoutingTable() map[string]string {
	return s.routingTable
}

func (s *staticConfig) GetCellEndpoints() map[string]string {
	return s.cellEndpoints
}

func (s *staticConfig) GetDefaultPlacement() string {
	return s.defaultPlacement
}
