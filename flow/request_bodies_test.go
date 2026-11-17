package flow

import (
	"encoding/json"
	"testing"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRequestBodiesSendZeroValues(t *testing.T) {
	healthCheck := compute.LoadBalancerHealthCheckOptions{TypeID: 2, Interval: new(10)}

	for _, tc := range []struct {
		name string
		body interface{}
		want string
	}{
		{
			name: "sticky session off is sent",
			body: compute.LoadBalancerPoolUpdateReq{
				BalancingAlgorithmID: intPointer(types.Int64Value(1)),
				StickySession:        boolPointer(types.BoolValue(false)),
			},
			want: `{"balancing_algorithm_id":1,"sticky_session":false}`,
		},
		{
			name: "unset pool fields and an unchanged health check are left out",
			body: compute.LoadBalancerPoolUpdateReq{
				CertificateID: intPointer(types.Int64Null()),
				StickySession: boolPointer(types.BoolNull()),
			},
			want: `{}`,
		},
		{
			name: "a changed health check is sent",
			body: compute.LoadBalancerPoolUpdateReq{HealthCheck: &healthCheck},
			want: `{"health_check":{"type_id":2,"interval":10}}`,
		},
		{
			name: "router made private",
			body: compute.RouterUpdateReq{Name: new("router"), Public: boolPointer(types.BoolValue(false))},
			want: `{"name":"router","public":false}`,
		},
		{
			name: "router public left alone",
			body: compute.RouterUpdateReq{Name: new("router"), Public: boolPointer(types.BoolNull())},
			want: `{"name":"router"}`,
		},
		{
			name: "icmp echo request, type 8 code 0",
			body: compute.SecurityGroupRuleCreateReq{
				Direction: "ingress",
				Protocol:  compute.ProtocolICMP,
				ICMPType:  intPointer(types.Int64Value(8)),
				ICMPCode:  intPointer(types.Int64Value(0)),
				IPRange:   new("0.0.0.0/0"),
			},
			want: `{"direction":"ingress","protocol":1,"icmp_type":8,"icmp_code":0,"ip_range":"0.0.0.0/0"}`,
		},
		{
			name: "icmp echo reply, type 0 code 0",
			body: compute.SecurityGroupRuleCreateReq{
				Direction: "ingress",
				Protocol:  compute.ProtocolICMP,
				ICMPType:  intPointer(types.Int64Value(0)),
				ICMPCode:  intPointer(types.Int64Value(0)),
			},
			want: `{"direction":"ingress","protocol":1,"icmp_type":0,"icmp_code":0}`,
		},
		{
			name: "tcp rule carries no icmp fields",
			body: compute.SecurityGroupRuleCreateReq{Direction: "ingress", Protocol: compute.ProtocolTCP, FromPort: new(22), ToPort: new(22)},
			want: `{"direction":"ingress","protocol":6,"from_port":22,"to_port":22}`,
		},
	} {
		got, err := json.Marshal(tc.body)
		if err != nil {
			t.Errorf("%s: marshal returned %s", tc.name, err)
			continue
		}
		if string(got) != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

func TestSameHealthCheck(t *testing.T) {
	state := &computeLoadBalancerHealthCheckResourceData{
		TypeID:             types.Int64Value(2),
		Interval:           types.StringValue("1m0s"),
		Timeout:            types.StringValue("5s"),
		HealthyThreshold:   types.Int64Value(3),
		UnhealthyThreshold: types.Int64Value(3),
	}
	planned := compute.LoadBalancerHealthCheckOptions{TypeID: 2, Interval: new(60), Timeout: new(5), HealthyThreshold: new(3), UnhealthyThreshold: new(3)}

	if !sameHealthCheck(state, planned) {
		t.Error("60s in the config and 1m0s in the state should count as the same health check")
	}

	planned.Interval = new(30)
	if sameHealthCheck(state, planned) {
		t.Error("a changed interval should count as a different health check")
	}

	if sameHealthCheck(nil, planned) {
		t.Error("no health check in the state should always send one")
	}
}
