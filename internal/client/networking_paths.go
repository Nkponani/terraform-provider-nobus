// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

const (
	subnetsPath             = "/api/v3/network/subnet"
	routersPath             = "/api/v3/network/router"
	routerAddInterfacePath  = "/api/v3/network/router/add-interface"
	routerDropInterfacePath = "/api/v3/network/router/remove-interface"
	portsPath               = "/api/v3/network/port"
	floatingIPsPath         = "/api/v3/network/floating-ip"
	securityGroupsPath      = "/api/v3/network/security-group"
	securityGroupRulesPath  = "/api/v3/network/security-group-rule"
)
const (
	trunksPath                 = "/api/v3/network/trunk"
	trunkListPath              = "/api/v3/network/trunk/list"
	subnetListPath             = "/api/v3/network/subnet/list"
	portListPath               = "/api/v3/network/port/list"
	routerListPath             = "/api/v3/network/router/list"
	floatingIPListPath         = "/api/v3/network/floating-ip/list"
	securityGroupListPath      = "/api/v3/network/security-group/list"
	securityGroupRuleListPath  = "/api/v3/network/security-group-rule/list"
	qosPoliciesPath            = "/api/v3/network/qos-policy"
	qosPolicyListPath          = "/api/v3/network/qos-policy/list"
	firewallGroupsPath         = "/api/v3/network/firewall-group"
	firewallGroupListPath      = "/api/v3/network/firewall-group/list"
	portAddressPairsPath       = "/api/v3/network/port/address-pairs"
	portClearAddressPairsPath  = "/api/v3/network/port/clear-address-pairs"
	routerClearGatewayPath     = "/api/v3/network/router/clear-gateway"
	floatingIPDisassociatePath = "/api/v3/network/floating-ip/disassociate"
	standaloneNetworksPath     = "/api/v3/network/only"
)
