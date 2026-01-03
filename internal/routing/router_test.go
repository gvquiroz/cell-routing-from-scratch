package routing

import (
	"testing"
)

func TestRouter_Route(t *testing.T) {
	routingTable := map[string]string{
		"acme":    "tier1",
		"globex":  "tier2",
		"initech": "tier3",
		"visa":    "visa",
	}

	cellEndpoints := map[string]string{
		"tier1": "http://cell-tier1:9001",
		"tier2": "http://cell-tier2:9002",
		"tier3": "http://cell-tier3:9003",
		"visa":  "http://cell-visa:9004",
	}

	router := NewRouter(routingTable, cellEndpoints, "tier3")

	tests := []struct {
		name          string
		routingKey    string
		wantPlacement string
		wantReason    RouteReason
		wantEndpoint  string
		wantErr       bool
	}{
		{
			name:          "dedicated customer",
			routingKey:    "visa",
			wantPlacement: "visa",
			wantReason:    ReasonDedicated,
			wantEndpoint:  "http://cell-visa:9004",
			wantErr:       false,
		},
		{
			name:          "tier1 customer",
			routingKey:    "acme",
			wantPlacement: "tier1",
			wantReason:    ReasonTier,
			wantEndpoint:  "http://cell-tier1:9001",
			wantErr:       false,
		},
		{
			name:          "tier2 customer",
			routingKey:    "globex",
			wantPlacement: "tier2",
			wantReason:    ReasonTier,
			wantEndpoint:  "http://cell-tier2:9002",
			wantErr:       false,
		},
		{
			name:          "tier3 customer",
			routingKey:    "initech",
			wantPlacement: "tier3",
			wantReason:    ReasonTier,
			wantEndpoint:  "http://cell-tier3:9003",
			wantErr:       false,
		},
		{
			name:          "missing routing key defaults to tier3",
			routingKey:    "",
			wantPlacement: "tier3",
			wantReason:    ReasonDefault,
			wantEndpoint:  "http://cell-tier3:9003",
			wantErr:       false,
		},
		{
			name:          "unknown routing key defaults to tier3",
			routingKey:    "unknown-customer",
			wantPlacement: "tier3",
			wantReason:    ReasonDefault,
			wantEndpoint:  "http://cell-tier3:9003",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := router.Route(tt.routingKey)

			if (err != nil) != tt.wantErr {
				t.Errorf("Route() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if decision.PlacementKey != tt.wantPlacement {
				t.Errorf("Route() placement = %v, want %v", decision.PlacementKey, tt.wantPlacement)
			}

			if decision.Reason != tt.wantReason {
				t.Errorf("Route() reason = %v, want %v", decision.Reason, tt.wantReason)
			}

			if decision.EndpointURL != tt.wantEndpoint {
				t.Errorf("Route() endpoint = %v, want %v", decision.EndpointURL, tt.wantEndpoint)
			}
		})
	}
}

func TestRouter_Route_MissingEndpoint(t *testing.T) {
	routingTable := map[string]string{
		"orphan": "orphan-placement",
	}

	cellEndpoints := map[string]string{
		"tier3": "http://cell-tier3:9003",
	}

	router := NewRouter(routingTable, cellEndpoints, "tier3")

	_, err := router.Route("orphan")
	if err == nil {
		t.Error("Route() expected error for missing endpoint, got nil")
	}
}
