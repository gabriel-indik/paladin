// Code generated by mockery v2.44.1. DO NOT EDIT.

package mocks

import (
	filters "github.com/kaleido-io/paladin/kata/internal/filters"
	mock "github.com/stretchr/testify/mock"
)

// WithValueSet is an autogenerated mock type for the WithValueSet type
type WithValueSet struct {
	mock.Mock
}

// ValueSet provides a mock function with given fields:
func (_m *WithValueSet) ValueSet() filters.ValueSet {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ValueSet")
	}

	var r0 filters.ValueSet
	if rf, ok := ret.Get(0).(func() filters.ValueSet); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(filters.ValueSet)
		}
	}

	return r0
}

// NewWithValueSet creates a new instance of WithValueSet. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewWithValueSet(t interface {
	mock.TestingT
	Cleanup(func())
}) *WithValueSet {
	mock := &WithValueSet{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
