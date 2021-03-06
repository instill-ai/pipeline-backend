// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha (interfaces: UserServiceClient)

// Package service_test is a generated GoMock package.
package service_test

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	mgmtv1alpha "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	grpc "google.golang.org/grpc"
)

// MockUserServiceClient is a mock of UserServiceClient interface.
type MockUserServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockUserServiceClientMockRecorder
}

// MockUserServiceClientMockRecorder is the mock recorder for MockUserServiceClient.
type MockUserServiceClientMockRecorder struct {
	mock *MockUserServiceClient
}

// NewMockUserServiceClient creates a new mock instance.
func NewMockUserServiceClient(ctrl *gomock.Controller) *MockUserServiceClient {
	mock := &MockUserServiceClient{ctrl: ctrl}
	mock.recorder = &MockUserServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserServiceClient) EXPECT() *MockUserServiceClientMockRecorder {
	return m.recorder
}

// CreateUser mocks base method.
func (m *MockUserServiceClient) CreateUser(arg0 context.Context, arg1 *mgmtv1alpha.CreateUserRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.CreateUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateUser", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.CreateUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockUserServiceClientMockRecorder) CreateUser(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockUserServiceClient)(nil).CreateUser), varargs...)
}

// DeleteUser mocks base method.
func (m *MockUserServiceClient) DeleteUser(arg0 context.Context, arg1 *mgmtv1alpha.DeleteUserRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.DeleteUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteUser", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.DeleteUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockUserServiceClientMockRecorder) DeleteUser(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockUserServiceClient)(nil).DeleteUser), varargs...)
}

// GetUser mocks base method.
func (m *MockUserServiceClient) GetUser(arg0 context.Context, arg1 *mgmtv1alpha.GetUserRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.GetUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetUser", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.GetUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserServiceClientMockRecorder) GetUser(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserServiceClient)(nil).GetUser), varargs...)
}

// ListUser mocks base method.
func (m *MockUserServiceClient) ListUser(arg0 context.Context, arg1 *mgmtv1alpha.ListUserRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.ListUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListUser", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.ListUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUser indicates an expected call of ListUser.
func (mr *MockUserServiceClientMockRecorder) ListUser(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUser", reflect.TypeOf((*MockUserServiceClient)(nil).ListUser), varargs...)
}

// Liveness mocks base method.
func (m *MockUserServiceClient) Liveness(arg0 context.Context, arg1 *mgmtv1alpha.LivenessRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.LivenessResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Liveness", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.LivenessResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Liveness indicates an expected call of Liveness.
func (mr *MockUserServiceClientMockRecorder) Liveness(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Liveness", reflect.TypeOf((*MockUserServiceClient)(nil).Liveness), varargs...)
}

// LookUpUser mocks base method.
func (m *MockUserServiceClient) LookUpUser(arg0 context.Context, arg1 *mgmtv1alpha.LookUpUserRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.LookUpUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "LookUpUser", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.LookUpUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LookUpUser indicates an expected call of LookUpUser.
func (mr *MockUserServiceClientMockRecorder) LookUpUser(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LookUpUser", reflect.TypeOf((*MockUserServiceClient)(nil).LookUpUser), varargs...)
}

// Readiness mocks base method.
func (m *MockUserServiceClient) Readiness(arg0 context.Context, arg1 *mgmtv1alpha.ReadinessRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.ReadinessResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Readiness", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.ReadinessResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Readiness indicates an expected call of Readiness.
func (mr *MockUserServiceClientMockRecorder) Readiness(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Readiness", reflect.TypeOf((*MockUserServiceClient)(nil).Readiness), varargs...)
}

// UpdateUser mocks base method.
func (m *MockUserServiceClient) UpdateUser(arg0 context.Context, arg1 *mgmtv1alpha.UpdateUserRequest, arg2 ...grpc.CallOption) (*mgmtv1alpha.UpdateUserResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateUser", varargs...)
	ret0, _ := ret[0].(*mgmtv1alpha.UpdateUserResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserServiceClientMockRecorder) UpdateUser(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserServiceClient)(nil).UpdateUser), varargs...)
}
