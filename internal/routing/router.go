package routing

import "fmt"

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
	routingTable     map[string]string
	cellEndpoints    map[string]string
	defaultPlacement string
}

// NewRouter creates a new Router with the given mappings
func NewRouter(
	routingTable map[string]string,
	cellEndpoints map[string]string,
	defaultPlacement string,
) *Router {
	return &Router{
		routingTable:     routingTable,
		cellEndpoints:    cellEndpoints,
		defaultPlacement: defaultPlacement,
	}
}

// Route determines the placement and endpoint for a given routing key
func (r *Router) Route(routingKey string) (*RoutingDecision, error) {
	// Lookup placement (use default if not found or empty)
	placementKey, found := r.routingTable[routingKey]
	if !found || routingKey == "" {
		placementKey = r.defaultPlacement
	}

	// Determine reason
	reason := r.determineReason(routingKey, found)

	// Lookup endpoint URL
	endpointURL, found := r.cellEndpoints[placementKey]
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
func (r *Router) determineReason(routingKey string, found bool) RouteReason {
	if !found || routingKey == "" {
		return ReasonDefault
	}

	placementKey := r.routingTable[routingKey]
	if r.isTier(placementKey) {
		return ReasonTier
	}
	return ReasonDedicated
}

// isTier checks if the placement key is a shared tier
func (r *Router) isTier(placementKey string) bool {
	return placementKey == "tier1" || placementKey == "tier2" || placementKey == "tier3"
}
