// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import account "github.com/Optum/dce/pkg/account"
import accountmanageriface "github.com/Optum/dce/pkg/accountmanager/accountmanageriface"
import arn "github.com/Optum/dce/pkg/arn"
import mock "github.com/stretchr/testify/mock"

// Servicer is an autogenerated mock type for the Servicer type
type Servicer struct {
	mock.Mock
}

// ConsoleURL provides a mock function with given fields: creds
func (_m *Servicer) ConsoleURL(creds accountmanageriface.Credentialer) (string, error) {
	ret := _m.Called(creds)

	var r0 string
	if rf, ok := ret.Get(0).(func(accountmanageriface.Credentialer) string); ok {
		r0 = rf(creds)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(accountmanageriface.Credentialer) error); ok {
		r1 = rf(creds)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Credentials provides a mock function with given fields: role, roleSessionName
func (_m *Servicer) Credentials(role *arn.ARN, roleSessionName string) accountmanageriface.Credentialer {
	ret := _m.Called(role, roleSessionName)

	var r0 accountmanageriface.Credentialer
	if rf, ok := ret.Get(0).(func(*arn.ARN, string) accountmanageriface.Credentialer); ok {
		r0 = rf(role, roleSessionName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(accountmanageriface.Credentialer)
		}
	}

	return r0
}

// DeletePrincipalAccess provides a mock function with given fields: _a0
func (_m *Servicer) DeletePrincipalAccess(_a0 *account.Account) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*account.Account) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpsertPrincipalAccess provides a mock function with given fields: _a0
func (_m *Servicer) UpsertPrincipalAccess(_a0 *account.Account) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*account.Account) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateAccess provides a mock function with given fields: role
func (_m *Servicer) ValidateAccess(role *arn.ARN) error {
	ret := _m.Called(role)

	var r0 error
	if rf, ok := ret.Get(0).(func(*arn.ARN) error); ok {
		r0 = rf(role)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
