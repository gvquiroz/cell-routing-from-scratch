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
	customerToPlacement map[string]string
	placementToEndpoint map[string]string
	defaultPlacement    string
}

// NewRouter creates a new Router with the given mappings
func NewRouter(
	customerToPlacement map[string]string,
	placementToEndpoint map[string]string,
	defaultPlacement string,
) *Router {
	return &Router{
		customerToPlacement: customerToPlacement,
		placementToEndpoint: placementToEndpoint,
		defaultPlacement:    defaultPlacement,
	}
}

// Route determines the placement and endpoint for a given routing key
func (r *Router) Route(routingKey string) (*RoutingDecision, error) {
	var placementKey string
	var reason RouteReason

	// Determine placement key and reason
	if routingKey == "" {
		// Missing routing key -> default
		placementKey = r.defaultPlacement
		reason = ReasonDefault
	} else if placement, found := r.customerToPlacement[routingKey]; found {
		// Routing key found
		placementKey = placement
		if r.isTier(placementKey) {
			reason = ReasonTier
		} else {
			reason = ReasonDedicated
		}
	} else {
		// Routing key not found -> default
		placementKey = r.defaultPlacement
		reason = ReasonDefault
	}

	// Lookup endpoint URL
	endpointURL, found := r.placementToEndpoint[placementKey]
	if !found {
		return nil, fmt.Errorf("no endpoint configured for placement: %s", placementKey)
	}

	return &RoutingDecision{
		PlacementKey: placementKey,
		Reason:       reason,
		EndpointURL:  endpointURL,
	}, nil
}

// isTier checks if the placement key is a shared tier
func (r *Router) isTier(placementKey string) bool {
	return placementKey == "tier1" || placementKey == "tier2" || placementKey == "tier3"
}
