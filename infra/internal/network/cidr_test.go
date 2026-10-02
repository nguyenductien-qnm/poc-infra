package network

import (
	"reflect"
	"testing"
)

func TestSubnetPlan(t *testing.T) {
	tests := []struct {
		vpcCidr     string
		wantPublic  []string
		wantPrivate []string
	}{
		{"10.10.0.0/16", []string{"10.10.1.0/24", "10.10.2.0/24"}, []string{"10.10.10.0/24", "10.10.20.0/24"}},
		{"10.30.0.0/16", []string{"10.30.1.0/24", "10.30.2.0/24"}, []string{"10.30.10.0/24", "10.30.20.0/24"}},
		{"172.16.32.0/19", []string{"172.16.33.0/24", "172.16.34.0/24"}, []string{"172.16.42.0/24", "172.16.52.0/24"}},
	}
	for _, tt := range tests {
		pub, priv, err := subnetPlan(tt.vpcCidr)
		if err != nil {
			t.Fatalf("subnetPlan(%q) error: %v", tt.vpcCidr, err)
		}
		if !reflect.DeepEqual(pub, tt.wantPublic) {
			t.Errorf("subnetPlan(%q) public = %v, want %v", tt.vpcCidr, pub, tt.wantPublic)
		}
		if !reflect.DeepEqual(priv, tt.wantPrivate) {
			t.Errorf("subnetPlan(%q) private = %v, want %v", tt.vpcCidr, priv, tt.wantPrivate)
		}
	}
}

func TestSubnetPlanErrors(t *testing.T) {
	for _, cidr := range []string{
		"10.10.0.0/20", // chi chua 16 subnet /24, index 20 vuot qua
		"10.10.0.0/25", // nho hon /24
		"not-a-cidr",
		"fd00::/48", // IPv6
	} {
		if _, _, err := subnetPlan(cidr); err == nil {
			t.Errorf("subnetPlan(%q) expected error, got nil", cidr)
		}
	}
}
