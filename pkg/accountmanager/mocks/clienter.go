// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import arn "github.com/Optum/dce/pkg/arn"
import aws "github.com/aws/aws-sdk-go/aws"
import iamiface "github.com/aws/aws-sdk-go/service/iam/iamiface"
import mock "github.com/stretchr/testify/mock"

// clienter is an autogenerated mock type for the clienter type
type clienter struct {
	mock.Mock
}

// Config provides a mock function with given fields: roleArn, roleSessionName
func (_m *clienter) Config(roleArn *arn.ARN, roleSessionName string) *aws.Config {
	ret := _m.Called(roleArn, roleSessionName)

	var r0 *aws.Config
	if rf, ok := ret.Get(0).(func(*arn.ARN, string) *aws.Config); ok {
		r0 = rf(roleArn, roleSessionName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*aws.Config)
		}
	}

	return r0
}

// IAM provides a mock function with given fields: roleArn
func (_m *clienter) IAM(roleArn *arn.ARN) iamiface.IAMAPI {
	ret := _m.Called(roleArn)

	var r0 iamiface.IAMAPI
	if rf, ok := ret.Get(0).(func(*arn.ARN) iamiface.IAMAPI); ok {
		r0 = rf(roleArn)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(iamiface.IAMAPI)
		}
	}

	return r0
}
