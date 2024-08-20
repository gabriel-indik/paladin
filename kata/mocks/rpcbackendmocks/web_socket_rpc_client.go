// Code generated by mockery v2.44.1. DO NOT EDIT.

package rpcbackendmocks

import (
	context "context"

	rpcbackend "github.com/hyperledger/firefly-signer/pkg/rpcbackend"
	mock "github.com/stretchr/testify/mock"
)

// WebSocketRPCClient is an autogenerated mock type for the WebSocketRPCClient type
type WebSocketRPCClient struct {
	mock.Mock
}

// CallRPC provides a mock function with given fields: ctx, result, method, params
func (_m *WebSocketRPCClient) CallRPC(ctx context.Context, result interface{}, method string, params ...interface{}) *rpcbackend.RPCError {
	var _ca []interface{}
	_ca = append(_ca, ctx, result, method)
	_ca = append(_ca, params...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for CallRPC")
	}

	var r0 *rpcbackend.RPCError
	if rf, ok := ret.Get(0).(func(context.Context, interface{}, string, ...interface{}) *rpcbackend.RPCError); ok {
		r0 = rf(ctx, result, method, params...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rpcbackend.RPCError)
		}
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *WebSocketRPCClient) Close() {
	_m.Called()
}

// Connect provides a mock function with given fields: ctx
func (_m *WebSocketRPCClient) Connect(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Connect")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Subscribe provides a mock function with given fields: ctx, params
func (_m *WebSocketRPCClient) Subscribe(ctx context.Context, params ...interface{}) (rpcbackend.Subscription, *rpcbackend.RPCError) {
	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, params...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Subscribe")
	}

	var r0 rpcbackend.Subscription
	var r1 *rpcbackend.RPCError
	if rf, ok := ret.Get(0).(func(context.Context, ...interface{}) (rpcbackend.Subscription, *rpcbackend.RPCError)); ok {
		return rf(ctx, params...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ...interface{}) rpcbackend.Subscription); ok {
		r0 = rf(ctx, params...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rpcbackend.Subscription)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ...interface{}) *rpcbackend.RPCError); ok {
		r1 = rf(ctx, params...)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*rpcbackend.RPCError)
		}
	}

	return r0, r1
}

// Subscriptions provides a mock function with given fields:
func (_m *WebSocketRPCClient) Subscriptions() []rpcbackend.Subscription {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Subscriptions")
	}

	var r0 []rpcbackend.Subscription
	if rf, ok := ret.Get(0).(func() []rpcbackend.Subscription); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]rpcbackend.Subscription)
		}
	}

	return r0
}

// UnsubscribeAll provides a mock function with given fields: ctx
func (_m *WebSocketRPCClient) UnsubscribeAll(ctx context.Context) *rpcbackend.RPCError {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UnsubscribeAll")
	}

	var r0 *rpcbackend.RPCError
	if rf, ok := ret.Get(0).(func(context.Context) *rpcbackend.RPCError); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rpcbackend.RPCError)
		}
	}

	return r0
}

// NewWebSocketRPCClient creates a new instance of WebSocketRPCClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewWebSocketRPCClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *WebSocketRPCClient {
	mock := &WebSocketRPCClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
