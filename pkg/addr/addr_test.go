package addr

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/linkerd/linkerd2-proxy-api/go/net"
	proxy "github.com/linkerd/linkerd2-proxy-api/go/net"
	"github.com/linkerd/linkerd2/controller/gen/public"
)

func TestPublicAddressToString(t *testing.T) {
	cases := []struct {
		name     string
		addr     *public.TcpAddress
		expected string
	}{
		{
			name: "ipv4",
			addr: &public.TcpAddress{
				Ip: &public.IPAddress{
					Ip: &public.IPAddress_Ipv4{
						Ipv4: 3232235521,
					},
				},
				Port: 1234,
			},
			expected: "192.168.0.1:1234",
		},
		{
			name: "ipv6",
			addr: &public.TcpAddress{
				Ip: &public.IPAddress{
					Ip: &public.IPAddress_Ipv6{
						Ipv6: &public.IPv6{
							First: 49320,
							Last:  1,
						},
					},
				},
				Port: 1234,
			},
			expected: "[::c0a8:0:0:0:1]:1234",
		},
		{
			name:     "nil",
			addr:     nil,
			expected: "<nil>:0",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := PublicAddressToString(c.addr)
			if c.expected != got {
				t.Errorf("expected: %v, got: %v", c.expected, got)
			}
		})
	}
}

func TestProxyAddressesToString(t *testing.T) {
	cases := []struct {
		name     string
		addrs    []pb.TcpAddress
		expected string
	}{
		{
			name: "ipv4",
			addrs: []pb.TcpAddress{
				{
					Ip: &proxy.IPAddress{
						Ip: &proxy.IPAddress_Ipv4{
							Ipv4: 3232235521,
						},
					},
					Port: 1234,
				},
				{
					Ip: &proxy.IPAddress{
						Ip: &proxy.IPAddress_Ipv4{
							Ipv4: 3232235522,
						},
					},
					Port: 1234,
				},
			},
			expected: "[192.168.0.1:1234,192.168.0.2:1234]",
		},
		{
			name:     "nil",
			addrs:    nil,
			expected: "[]",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := ProxyAddressesToString(c.addrs)
			if c.expected != got {
				t.Errorf("expected: %v, got: %v", c.expected, got)
			}
		})
	}
}

func TestProxyIPToString(t *testing.T) {
	cases := []struct {
		name     string
		ip       *pb.IPAddress
		expected string
	}{
		{
			name: "ipv4",
			ip: &pb.IPAddress{
				Ip: &pb.IPAddress_Ipv4{
					Ipv4: 3232235521,
				},
			},
			expected: "192.168.0.1",
		},
		{
			name:     "nil",
			ip:       nil,
			expected: "0.0.0.0",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := ProxyIPToString(c.ip)
			if c.expected != got {
				t.Errorf("expected: %v, got: %v", c.expected, got)
			}
		})
	}
}

func TestNetToPublic(t *testing.T) {

	type addrExp struct {
		proxyAddr     *proxy.TcpAddress
		publicAddress *public.TcpAddress
	}

	expectations := []addrExp{
		{
			proxyAddr:     &proxy.TcpAddress{},
			publicAddress: &public.TcpAddress{},
		},
		{
			proxyAddr: &proxy.TcpAddress{
				Ip:   &proxy.IPAddress{Ip: &proxy.IPAddress_Ipv4{Ipv4: 1}},
				Port: 1234,
			},
			publicAddress: &public.TcpAddress{
				Ip:   &public.IPAddress{Ip: &public.IPAddress_Ipv4{Ipv4: 1}},
				Port: 1234,
			},
		},
		{
			proxyAddr: &proxy.TcpAddress{
				Ip: &proxy.IPAddress{
					Ip: &proxy.IPAddress_Ipv6{
						Ipv6: &proxy.IPv6{
							First: 2345,
							Last:  6789,
						},
					},
				},
				Port: 1234,
			},
			publicAddress: &public.TcpAddress{
				Ip: &public.IPAddress{
					Ip: &public.IPAddress_Ipv6{
						Ipv6: &public.IPv6{
							First: 2345,
							Last:  6789,
						},
					},
				},
				Port: 1234,
			},
		},
	}

	for i, exp := range expectations {
		exp := exp // pin
		t.Run(fmt.Sprintf("%d returns expected public API TCPAddress", i), func(t *testing.T) {
			res := NetToPublic(exp.proxyAddr)
			if !proto.Equal(res, exp.publicAddress) {
				t.Fatalf("Unexpected TCP Address: [%+v] expected: [%+v]", res, exp.publicAddress)
			}
		})
	}
}
