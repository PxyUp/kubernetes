/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	hcservice "google.golang.org/grpc/health/grpc_health_v1"
	"k8s.io/kubernetes/pkg/probe"
	"net"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("Should: implement Probe interface", func(t *testing.T) {
		s := New()
		assert.Implements(t, (*Prober)(nil), s)
	})
}

type successServerMock struct {
}

func (s successServerMock) Check(context.Context, *hcservice.HealthCheckRequest) (*hcservice.HealthCheckResponse, error) {
	return &hcservice.HealthCheckResponse{
		Status: hcservice.HealthCheckResponse_SERVING,
	}, nil
}

func (s successServerMock) Watch(*hcservice.HealthCheckRequest, hcservice.Health_WatchServer) error {
	panic("implement me")
}

type errorTimeoutServerMock struct {
}

func (e errorTimeoutServerMock) Check(context.Context, *hcservice.HealthCheckRequest) (*hcservice.HealthCheckResponse, error) {
	time.Sleep(time.Second * 4)
	return &hcservice.HealthCheckResponse{
		Status: hcservice.HealthCheckResponse_SERVING,
	}, nil
}

func (e errorTimeoutServerMock) Watch(*hcservice.HealthCheckRequest, hcservice.Health_WatchServer) error {
	panic("implement me")
}

type errorNotServeServerMock struct {
}

func (e errorNotServeServerMock) Check(context.Context, *hcservice.HealthCheckRequest) (*hcservice.HealthCheckResponse, error) {
	return &hcservice.HealthCheckResponse{
		Status: hcservice.HealthCheckResponse_NOT_SERVING,
	}, nil
}

func (e errorNotServeServerMock) Watch(*hcservice.HealthCheckRequest, hcservice.Health_WatchServer) error {
	panic("implement me")
}

func TestGrpcProber_Probe(t *testing.T) {
	t.Run("Should: return error because cant find host", func(t *testing.T) {
		s := New()
		p, _, err := s.Probe("", 32, time.Second, grpc.WithInsecure(), grpc.WithBlock())
		assert.Equal(t, probe.Failure, p)
		assert.NotEqual(t, nil, err)
	})
	t.Run("Should: return error because server response not served", func(t *testing.T) {
		s := New()
		lis, _ := net.Listen("tcp", ":10413")
		grpcServer := grpc.NewServer()
		defer grpcServer.Stop()
		hcservice.RegisterHealthServer(grpcServer, &errorNotServeServerMock{})
		go func() {
			_ = grpcServer.Serve(lis)
		}()
		// take some time to wait server boot
		time.Sleep(2 * time.Second)
		p, _, err := s.Probe("0.0.0.0", 10413, time.Second, grpc.WithInsecure())
		assert.Equal(t, probe.Failure, p)
		assert.NotEqual(t, nil, err)
	})
	t.Run("Should: return error because server not response in time", func(t *testing.T) {
		s := New()
		lis, _ := net.Listen("tcp", ":10414")
		grpcServer := grpc.NewServer()
		defer grpcServer.Stop()
		hcservice.RegisterHealthServer(grpcServer, &errorTimeoutServerMock{})
		go func() {
			_ = grpcServer.Serve(lis)
		}()
		// take some time to wait server boot
		time.Sleep(2 * time.Second)
		p, _, err := s.Probe("0.0.0.0", 10414, time.Second*2, grpc.WithInsecure())
		assert.Equal(t, probe.Failure, p)
		assert.NotEqual(t, nil, err)
	})
	t.Run("Should: not return error because check was success", func(t *testing.T) {
		s := New()
		lis, _ := net.Listen("tcp", ":10415")
		grpcServer := grpc.NewServer()
		defer grpcServer.Stop()
		hcservice.RegisterHealthServer(grpcServer, &successServerMock{})
		go func() {
			_ = grpcServer.Serve(lis)
		}()
		// take some time to wait server boot
		time.Sleep(2 * time.Second)
		p, _, err := s.Probe("0.0.0.0", 10415, time.Second*2, grpc.WithInsecure())
		assert.Equal(t, probe.Success, p)
		assert.Equal(t, nil, err)
	})
}
