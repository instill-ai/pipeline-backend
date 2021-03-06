// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/instill-ai/protogen-go/vdp/connector/v1alpha (interfaces: ConnectorServiceClient)

// Package service_test is a generated GoMock package.
package service_test

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	connectorv1alpha "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	grpc "google.golang.org/grpc"
)

// MockConnectorServiceClient is a mock of ConnectorServiceClient interface.
type MockConnectorServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockConnectorServiceClientMockRecorder
}

// MockConnectorServiceClientMockRecorder is the mock recorder for MockConnectorServiceClient.
type MockConnectorServiceClientMockRecorder struct {
	mock *MockConnectorServiceClient
}

// NewMockConnectorServiceClient creates a new mock instance.
func NewMockConnectorServiceClient(ctrl *gomock.Controller) *MockConnectorServiceClient {
	mock := &MockConnectorServiceClient{ctrl: ctrl}
	mock.recorder = &MockConnectorServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConnectorServiceClient) EXPECT() *MockConnectorServiceClientMockRecorder {
	return m.recorder
}

// ConnectDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) ConnectDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.ConnectDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ConnectDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ConnectDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ConnectDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConnectDestinationConnector indicates an expected call of ConnectDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) ConnectDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).ConnectDestinationConnector), varargs...)
}

// ConnectSourceConnector mocks base method.
func (m *MockConnectorServiceClient) ConnectSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.ConnectSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ConnectSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ConnectSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ConnectSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConnectSourceConnector indicates an expected call of ConnectSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) ConnectSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).ConnectSourceConnector), varargs...)
}

// CreateDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) CreateDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.CreateDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.CreateDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.CreateDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateDestinationConnector indicates an expected call of CreateDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) CreateDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).CreateDestinationConnector), varargs...)
}

// CreateSourceConnector mocks base method.
func (m *MockConnectorServiceClient) CreateSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.CreateSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.CreateSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.CreateSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSourceConnector indicates an expected call of CreateSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) CreateSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).CreateSourceConnector), varargs...)
}

// DeleteDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) DeleteDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.DeleteDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.DeleteDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.DeleteDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteDestinationConnector indicates an expected call of DeleteDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) DeleteDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).DeleteDestinationConnector), varargs...)
}

// DeleteSourceConnector mocks base method.
func (m *MockConnectorServiceClient) DeleteSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.DeleteSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.DeleteSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.DeleteSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteSourceConnector indicates an expected call of DeleteSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) DeleteSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).DeleteSourceConnector), varargs...)
}

// DisconnectDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) DisconnectDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.DisconnectDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.DisconnectDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DisconnectDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.DisconnectDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DisconnectDestinationConnector indicates an expected call of DisconnectDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) DisconnectDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DisconnectDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).DisconnectDestinationConnector), varargs...)
}

// DisconnectSourceConnector mocks base method.
func (m *MockConnectorServiceClient) DisconnectSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.DisconnectSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.DisconnectSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DisconnectSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.DisconnectSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DisconnectSourceConnector indicates an expected call of DisconnectSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) DisconnectSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DisconnectSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).DisconnectSourceConnector), varargs...)
}

// GetDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) GetDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.GetDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.GetDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.GetDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDestinationConnector indicates an expected call of GetDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) GetDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).GetDestinationConnector), varargs...)
}

// GetDestinationConnectorDefinition mocks base method.
func (m *MockConnectorServiceClient) GetDestinationConnectorDefinition(arg0 context.Context, arg1 *connectorv1alpha.GetDestinationConnectorDefinitionRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.GetDestinationConnectorDefinitionResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetDestinationConnectorDefinition", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.GetDestinationConnectorDefinitionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDestinationConnectorDefinition indicates an expected call of GetDestinationConnectorDefinition.
func (mr *MockConnectorServiceClientMockRecorder) GetDestinationConnectorDefinition(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDestinationConnectorDefinition", reflect.TypeOf((*MockConnectorServiceClient)(nil).GetDestinationConnectorDefinition), varargs...)
}

// GetSourceConnector mocks base method.
func (m *MockConnectorServiceClient) GetSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.GetSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.GetSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.GetSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSourceConnector indicates an expected call of GetSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) GetSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).GetSourceConnector), varargs...)
}

// GetSourceConnectorDefinition mocks base method.
func (m *MockConnectorServiceClient) GetSourceConnectorDefinition(arg0 context.Context, arg1 *connectorv1alpha.GetSourceConnectorDefinitionRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.GetSourceConnectorDefinitionResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetSourceConnectorDefinition", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.GetSourceConnectorDefinitionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSourceConnectorDefinition indicates an expected call of GetSourceConnectorDefinition.
func (mr *MockConnectorServiceClientMockRecorder) GetSourceConnectorDefinition(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSourceConnectorDefinition", reflect.TypeOf((*MockConnectorServiceClient)(nil).GetSourceConnectorDefinition), varargs...)
}

// ListDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) ListDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.ListDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ListDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ListDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDestinationConnector indicates an expected call of ListDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) ListDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).ListDestinationConnector), varargs...)
}

// ListDestinationConnectorDefinition mocks base method.
func (m *MockConnectorServiceClient) ListDestinationConnectorDefinition(arg0 context.Context, arg1 *connectorv1alpha.ListDestinationConnectorDefinitionRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ListDestinationConnectorDefinitionResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListDestinationConnectorDefinition", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ListDestinationConnectorDefinitionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListDestinationConnectorDefinition indicates an expected call of ListDestinationConnectorDefinition.
func (mr *MockConnectorServiceClientMockRecorder) ListDestinationConnectorDefinition(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListDestinationConnectorDefinition", reflect.TypeOf((*MockConnectorServiceClient)(nil).ListDestinationConnectorDefinition), varargs...)
}

// ListSourceConnector mocks base method.
func (m *MockConnectorServiceClient) ListSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.ListSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ListSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ListSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListSourceConnector indicates an expected call of ListSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) ListSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).ListSourceConnector), varargs...)
}

// ListSourceConnectorDefinition mocks base method.
func (m *MockConnectorServiceClient) ListSourceConnectorDefinition(arg0 context.Context, arg1 *connectorv1alpha.ListSourceConnectorDefinitionRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ListSourceConnectorDefinitionResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListSourceConnectorDefinition", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ListSourceConnectorDefinitionResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListSourceConnectorDefinition indicates an expected call of ListSourceConnectorDefinition.
func (mr *MockConnectorServiceClientMockRecorder) ListSourceConnectorDefinition(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSourceConnectorDefinition", reflect.TypeOf((*MockConnectorServiceClient)(nil).ListSourceConnectorDefinition), varargs...)
}

// Liveness mocks base method.
func (m *MockConnectorServiceClient) Liveness(arg0 context.Context, arg1 *connectorv1alpha.LivenessRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.LivenessResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Liveness", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.LivenessResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Liveness indicates an expected call of Liveness.
func (mr *MockConnectorServiceClientMockRecorder) Liveness(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Liveness", reflect.TypeOf((*MockConnectorServiceClient)(nil).Liveness), varargs...)
}

// LookUpDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) LookUpDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.LookUpDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.LookUpDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "LookUpDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.LookUpDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LookUpDestinationConnector indicates an expected call of LookUpDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) LookUpDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LookUpDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).LookUpDestinationConnector), varargs...)
}

// LookUpSourceConnector mocks base method.
func (m *MockConnectorServiceClient) LookUpSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.LookUpSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.LookUpSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "LookUpSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.LookUpSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LookUpSourceConnector indicates an expected call of LookUpSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) LookUpSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LookUpSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).LookUpSourceConnector), varargs...)
}

// ReadSourceConnector mocks base method.
func (m *MockConnectorServiceClient) ReadSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.ReadSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ReadSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ReadSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ReadSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadSourceConnector indicates an expected call of ReadSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) ReadSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).ReadSourceConnector), varargs...)
}

// Readiness mocks base method.
func (m *MockConnectorServiceClient) Readiness(arg0 context.Context, arg1 *connectorv1alpha.ReadinessRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.ReadinessResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Readiness", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.ReadinessResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Readiness indicates an expected call of Readiness.
func (mr *MockConnectorServiceClientMockRecorder) Readiness(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Readiness", reflect.TypeOf((*MockConnectorServiceClient)(nil).Readiness), varargs...)
}

// RenameDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) RenameDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.RenameDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.RenameDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RenameDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.RenameDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RenameDestinationConnector indicates an expected call of RenameDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) RenameDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RenameDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).RenameDestinationConnector), varargs...)
}

// RenameSourceConnector mocks base method.
func (m *MockConnectorServiceClient) RenameSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.RenameSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.RenameSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RenameSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.RenameSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RenameSourceConnector indicates an expected call of RenameSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) RenameSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RenameSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).RenameSourceConnector), varargs...)
}

// UpdateDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) UpdateDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.UpdateDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.UpdateDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.UpdateDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateDestinationConnector indicates an expected call of UpdateDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) UpdateDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).UpdateDestinationConnector), varargs...)
}

// UpdateSourceConnector mocks base method.
func (m *MockConnectorServiceClient) UpdateSourceConnector(arg0 context.Context, arg1 *connectorv1alpha.UpdateSourceConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.UpdateSourceConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateSourceConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.UpdateSourceConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateSourceConnector indicates an expected call of UpdateSourceConnector.
func (mr *MockConnectorServiceClientMockRecorder) UpdateSourceConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateSourceConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).UpdateSourceConnector), varargs...)
}

// WriteDestinationConnector mocks base method.
func (m *MockConnectorServiceClient) WriteDestinationConnector(arg0 context.Context, arg1 *connectorv1alpha.WriteDestinationConnectorRequest, arg2 ...grpc.CallOption) (*connectorv1alpha.WriteDestinationConnectorResponse, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "WriteDestinationConnector", varargs...)
	ret0, _ := ret[0].(*connectorv1alpha.WriteDestinationConnectorResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WriteDestinationConnector indicates an expected call of WriteDestinationConnector.
func (mr *MockConnectorServiceClientMockRecorder) WriteDestinationConnector(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteDestinationConnector", reflect.TypeOf((*MockConnectorServiceClient)(nil).WriteDestinationConnector), varargs...)
}
