package network

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Vi tri cac subnet /24 trong VPC, moi phan tu ung voi 1 AZ.
// Vi du VPC 10.10.0.0/16: public 10.10.1.0/24, 10.10.2.0/24; private 10.10.10.0/24, 10.10.20.0/24.
var (
	publicSubnetIndexes  = []int{1, 2}
	privateSubnetIndexes = []int{10, 20}
)

// subnetPlan chia VPC CIDR thanh cac subnet /24 cho public va private.
func subnetPlan(vpcCidr string) (public, private []string, err error) {
	public, err = carveSubnets(vpcCidr, 24, publicSubnetIndexes)
	if err != nil {
		return nil, nil, err
	}
	private, err = carveSubnets(vpcCidr, 24, privateSubnetIndexes)
	if err != nil {
		return nil, nil, err
	}
	return public, private, nil
}

// carveSubnets tra ve cac subnet /newPrefix thu index trong vpcCidr (tuong tu cidrsubnet() cua Terraform).
func carveSubnets(vpcCidr string, newPrefix int, indexes []int) ([]string, error) {
	_, vpcNet, err := net.ParseCIDR(vpcCidr)
	if err != nil {
		return nil, fmt.Errorf("parsing vpc cidr %q: %w", vpcCidr, err)
	}
	vpcPrefix, bits := vpcNet.Mask.Size()
	if bits != 32 {
		return nil, fmt.Errorf("vpc cidr %q: only IPv4 is supported", vpcCidr)
	}
	if newPrefix < vpcPrefix {
		return nil, fmt.Errorf("vpc cidr %q is smaller than /%d", vpcCidr, newPrefix)
	}

	maxSubnets := 1 << (newPrefix - vpcPrefix)
	base := binary.BigEndian.Uint32(vpcNet.IP.To4())
	subnetSize := uint32(1) << (32 - newPrefix)

	cidrs := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		if idx < 0 || idx >= maxSubnets {
			return nil, fmt.Errorf("vpc cidr %q only fits %d subnets of /%d, index %d is out of range", vpcCidr, maxSubnets, newPrefix, idx)
		}
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, base+uint32(idx)*subnetSize)
		cidrs = append(cidrs, fmt.Sprintf("%s/%d", ip, newPrefix))
	}
	return cidrs, nil
}
