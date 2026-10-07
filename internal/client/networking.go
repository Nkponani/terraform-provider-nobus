// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

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

type Subnet struct {
	ID             string   `json:"id"`
	Name           *string  `json:"name"`
	Description    *string  `json:"description"`
	ProjectID      *string  `json:"project_id"`
	NetworkID      string   `json:"network_id"`
	CIDR           string   `json:"cidr"`
	IPVersion      int64    `json:"ip_version"`
	GatewayIP      *string  `json:"gateway_ip"`
	EnableDHCP     *bool    `json:"enable_dhcp"`
	DNSNameservers []string `json:"dns_nameservers"`
	Tags           []string `json:"tags"`
}

type SubnetCreate struct {
	NetworkID      string   `json:"network_id"`
	CIDR           string   `json:"cidr"`
	IPVersion      int64    `json:"ip_version"`
	Name           *string  `json:"name,omitempty"`
	Description    *string  `json:"description,omitempty"`
	GatewayIP      *string  `json:"gateway_ip,omitempty"`
	EnableDHCP     *bool    `json:"enable_dhcp,omitempty"`
	DNSNameservers []string `json:"dns_nameservers,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

type Router struct {
	ID           string         `json:"id"`
	Name         *string        `json:"name"`
	Description  *string        `json:"description"`
	ProjectID    *string        `json:"project_id"`
	AdminStateUp *bool          `json:"admin_state_up"`
	Status       *string        `json:"status"`
	Tags         []string       `json:"tags"`
	Gateway      map[string]any `json:"external_gateway_info"`
}

type RouterCreate struct {
	Name         string   `json:"name"`
	Description  *string  `json:"description,omitempty"`
	AdminStateUp *bool    `json:"admin_state_up,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type Port struct {
	ID             string   `json:"id"`
	Name           *string  `json:"name"`
	Description    *string  `json:"description"`
	ProjectID      *string  `json:"project_id"`
	NetworkID      string   `json:"network_id"`
	AdminStateUp   *bool    `json:"admin_state_up"`
	Status         *string  `json:"status"`
	MACAddress     *string  `json:"mac_address"`
	DeviceID       *string  `json:"device_id"`
	DeviceOwner    *string  `json:"device_owner"`
	SecurityGroups []string `json:"security_groups"`
	PortSecurity   *bool    `json:"port_security_enabled"`
	DNSName        *string  `json:"dns_name"`
}

type PortCreate struct {
	NetworkID      string   `json:"network_id"`
	Name           *string  `json:"name,omitempty"`
	Description    *string  `json:"description,omitempty"`
	AdminStateUp   *bool    `json:"admin_state_up,omitempty"`
	DeviceID       *string  `json:"device_id,omitempty"`
	DeviceOwner    *string  `json:"device_owner,omitempty"`
	SecurityGroups []string `json:"security_groups,omitempty"`
	PortSecurity   *bool    `json:"port_security_enabled,omitempty"`
	DNSName        *string  `json:"dns_name,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

type FloatingIP struct {
	ID                string   `json:"id"`
	Description       *string  `json:"description"`
	ProjectID         *string  `json:"project_id"`
	FloatingNetworkID string   `json:"floating_network_id"`
	FloatingIPAddress *string  `json:"floating_ip_address"`
	PortID            *string  `json:"port_id"`
	FixedIPAddress    *string  `json:"fixed_ip_address"`
	Status            *string  `json:"status"`
	Tags              []string `json:"tags"`
}

type FloatingIPCreate struct {
	FloatingNetworkID string   `json:"floating_network_id"`
	PortID            *string  `json:"port_id,omitempty"`
	FixedIPAddress    *string  `json:"fixed_ip_address,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}

type SecurityGroup struct {
	ID          string              `json:"id"`
	Name        *string             `json:"name"`
	Description *string             `json:"description"`
	ProjectID   *string             `json:"project_id"`
	Tags        []string            `json:"tags"`
	Rules       []SecurityGroupRule `json:"security_group_rules"`
}

type SecurityGroupRule struct {
	ID              string  `json:"id"`
	Description     *string `json:"description"`
	ProjectID       *string `json:"project_id"`
	SecurityGroupID string  `json:"security_group_id"`
	Direction       string  `json:"direction"`
	EtherType       string  `json:"ethertype"`
	Protocol        *string `json:"protocol"`
	PortRangeMin    *int64  `json:"port_range_min"`
	PortRangeMax    *int64  `json:"port_range_max"`
	RemoteIPPrefix  *string `json:"remote_ip_prefix"`
	RemoteGroupID   *string `json:"remote_group_id"`
}

type SecurityGroupCreate struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
type SecurityGroupRuleCreate struct {
	SecurityGroupID string  `json:"security_group_id"`
	Direction       string  `json:"direction"`
	EtherType       string  `json:"ethertype"`
	Protocol        *string `json:"protocol,omitempty"`
	PortRangeMin    *int64  `json:"port_range_min,omitempty"`
	PortRangeMax    *int64  `json:"port_range_max,omitempty"`
	RemoteIPPrefix  *string `json:"remote_ip_prefix,omitempty"`
	RemoteGroupID   *string `json:"remote_group_id,omitempty"`
	Description     *string `json:"description,omitempty"`
}

type scopedBody[T any] struct {
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
	Object           T      `json:"-"`
}

func networkQuery(projectID, availabilityZone, idKey, id string) map[string]string {
	return map[string]string{"project_id": projectID, "availability_zone": availabilityZone, idKey: id}
}

func (c *Client) networkObject(ctx context.Context, method, endpoint string, query map[string]string, body any, result any) error {
	var raw []byte
	var envelope struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := c.Do(ctx, Options{Method: method, Path: endpoint, Query: query, Body: body, Result: &envelope, RawBody: &raw}); err != nil {
		return err
	}
	if err := CheckStatus(method, endpoint, envelope.Status, raw); err != nil {
		return err
	}
	if result != nil {
		return json.Unmarshal(envelope.Data, result)
	}
	return nil
}

func (c *Client) CreateSubnet(ctx context.Context, p, z string, v SubnetCreate) (*Subnet, error) {
	body := struct {
		ProjectID        string       `json:"project_id"`
		AvailabilityZone string       `json:"availability_zone"`
		Subnet           SubnetCreate `json:"subnet"`
	}{p, z, v}
	var d struct {
		Subnet Subnet `json:"subnet"`
	}
	e := c.networkObject(ctx, http.MethodPost, subnetsPath, nil, body, &d)
	return &d.Subnet, e
}
func (c *Client) GetSubnet(ctx context.Context, p, z, id string) (*Subnet, error) {
	var d struct {
		Subnet Subnet `json:"subnet"`
	}
	e := c.networkObject(ctx, http.MethodGet, subnetsPath, networkQuery(p, z, "subnet_id", id), nil, &d)
	return &d.Subnet, e
}
func (c *Client) DeleteSubnet(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, subnetsPath, networkQuery(p, z, "subnet_id", id), nil, nil)
}
func (c *Client) CreateRouter(ctx context.Context, p, z string, v RouterCreate) (*Router, error) {
	body := struct {
		ProjectID        string       `json:"project_id"`
		AvailabilityZone string       `json:"availability_zone"`
		Router           RouterCreate `json:"router"`
	}{p, z, v}
	var d struct {
		Router Router `json:"router"`
	}
	e := c.networkObject(ctx, http.MethodPost, routersPath, nil, body, &d)
	return &d.Router, e
}
func (c *Client) GetRouter(ctx context.Context, p, z, id string) (*Router, error) {
	var d struct {
		Router Router `json:"router"`
	}
	e := c.networkObject(ctx, http.MethodGet, routersPath, networkQuery(p, z, "router_id", id), nil, &d)
	return &d.Router, e
}
func (c *Client) DeleteRouter(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, routersPath, networkQuery(p, z, "router_id", id), nil, nil)
}
func (c *Client) SetRouterInterface(ctx context.Context, p, z, routerID, subnetID, portID string, add bool) error {
	q := networkQuery(p, z, "router_id", routerID)
	q["subnet_id"] = subnetID
	q["port_id"] = portID
	if add {
		return c.networkObject(ctx, http.MethodPost, routerAddInterfacePath, q, nil, nil)
	}
	return c.networkObject(ctx, http.MethodDelete, routerDropInterfacePath, q, nil, nil)
}
func (c *Client) CreatePort(ctx context.Context, p, z string, v PortCreate) (*Port, error) {
	body := struct {
		ProjectID        string     `json:"project_id"`
		AvailabilityZone string     `json:"availability_zone"`
		Port             PortCreate `json:"port"`
	}{p, z, v}
	var d struct {
		Port Port `json:"port"`
	}
	e := c.networkObject(ctx, http.MethodPost, portsPath, nil, body, &d)
	return &d.Port, e
}
func (c *Client) GetPort(ctx context.Context, p, z, id string) (*Port, error) {
	var d struct {
		Port Port `json:"port"`
	}
	e := c.networkObject(ctx, http.MethodGet, portsPath, networkQuery(p, z, "port_id", id), nil, &d)
	return &d.Port, e
}
func (c *Client) DeletePort(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, portsPath, networkQuery(p, z, "port_id", id), nil, nil)
}
func (c *Client) CreateFloatingIP(ctx context.Context, p, z string, v FloatingIPCreate) (*FloatingIP, error) {
	body := struct {
		ProjectID        string           `json:"project_id"`
		AvailabilityZone string           `json:"availability_zone"`
		FloatingIP       FloatingIPCreate `json:"floatingip"`
	}{p, z, v}
	var d struct {
		FloatingIP FloatingIP `json:"floatingip"`
	}
	e := c.networkObject(ctx, http.MethodPost, floatingIPsPath, nil, body, &d)
	return &d.FloatingIP, e
}
func (c *Client) GetFloatingIP(ctx context.Context, p, z, id string) (*FloatingIP, error) {
	var d struct {
		FloatingIP FloatingIP `json:"floatingip"`
	}
	e := c.networkObject(ctx, http.MethodGet, floatingIPsPath, networkQuery(p, z, "floating_ip_id", id), nil, &d)
	return &d.FloatingIP, e
}
func (c *Client) DeleteFloatingIP(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, floatingIPsPath, networkQuery(p, z, "floating_ip_id", id), nil, nil)
}
func (c *Client) CreateSecurityGroup(ctx context.Context, p, z string, v SecurityGroupCreate) (*SecurityGroup, error) {
	body := struct {
		ProjectID        string              `json:"project_id"`
		AvailabilityZone string              `json:"availability_zone"`
		SecurityGroup    SecurityGroupCreate `json:"security_group"`
	}{p, z, v}
	var d struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	e := c.networkObject(ctx, http.MethodPost, securityGroupsPath, nil, body, &d)
	return &d.SecurityGroup, e
}
func (c *Client) GetSecurityGroup(ctx context.Context, p, z, id string) (*SecurityGroup, error) {
	var d struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	e := c.networkObject(ctx, http.MethodGet, securityGroupsPath, networkQuery(p, z, "security_group_id", id), nil, &d)
	return &d.SecurityGroup, e
}
func (c *Client) DeleteSecurityGroup(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, securityGroupsPath, networkQuery(p, z, "security_group_id", id), nil, nil)
}
func (c *Client) CreateSecurityGroupRule(ctx context.Context, p, z string, v SecurityGroupRuleCreate) (*SecurityGroupRule, error) {
	body := struct {
		ProjectID        string                  `json:"project_id"`
		AvailabilityZone string                  `json:"availability_zone"`
		Rule             SecurityGroupRuleCreate `json:"security_group_rule"`
	}{p, z, v}
	var d struct {
		Rule SecurityGroupRule `json:"security_group_rule"`
	}
	e := c.networkObject(ctx, http.MethodPost, securityGroupRulesPath, nil, body, &d)
	return &d.Rule, e
}
func (c *Client) GetSecurityGroupRule(ctx context.Context, p, z, id string) (*SecurityGroupRule, error) {
	var d struct {
		Rule SecurityGroupRule `json:"security_group_rule"`
	}
	e := c.networkObject(ctx, http.MethodGet, securityGroupRulesPath, networkQuery(p, z, "security_group_rule_id", id), nil, &d)
	return &d.Rule, e
}
func (c *Client) DeleteSecurityGroupRule(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, securityGroupRulesPath, networkQuery(p, z, "security_group_rule_id", id), nil, nil)
}
