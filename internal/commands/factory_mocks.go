// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/teserakt-io/c2/internal/commands (interfaces: Factory)

// Package commands is a generated GoMock package.
package commands

import (
	gomock "github.com/golang/mock/gomock"
	ed25519 "golang.org/x/crypto/ed25519"
	reflect "reflect"
)

// MockFactory is a mock of Factory interface
type MockFactory struct {
	ctrl     *gomock.Controller
	recorder *MockFactoryMockRecorder
}

// MockFactoryMockRecorder is the mock recorder for MockFactory
type MockFactoryMockRecorder struct {
	mock *MockFactory
}

// NewMockFactory creates a new mock instance
func NewMockFactory(ctrl *gomock.Controller) *MockFactory {
	mock := &MockFactory{ctrl: ctrl}
	mock.recorder = &MockFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFactory) EXPECT() *MockFactoryMockRecorder {
	return m.recorder
}

// CreateRemovePubKeyCommand mocks base method
func (m *MockFactory) CreateRemovePubKeyCommand(arg0 string) (Command, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRemovePubKeyCommand", arg0)
	ret0, _ := ret[0].(Command)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRemovePubKeyCommand indicates an expected call of CreateRemovePubKeyCommand
func (mr *MockFactoryMockRecorder) CreateRemovePubKeyCommand(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRemovePubKeyCommand", reflect.TypeOf((*MockFactory)(nil).CreateRemovePubKeyCommand), arg0)
}

// CreateRemoveTopicCommand mocks base method
func (m *MockFactory) CreateRemoveTopicCommand(arg0 []byte) (Command, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRemoveTopicCommand", arg0)
	ret0, _ := ret[0].(Command)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRemoveTopicCommand indicates an expected call of CreateRemoveTopicCommand
func (mr *MockFactoryMockRecorder) CreateRemoveTopicCommand(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRemoveTopicCommand", reflect.TypeOf((*MockFactory)(nil).CreateRemoveTopicCommand), arg0)
}

// CreateResetPubKeysCommand mocks base method
func (m *MockFactory) CreateResetPubKeysCommand() (Command, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateResetPubKeysCommand")
	ret0, _ := ret[0].(Command)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateResetPubKeysCommand indicates an expected call of CreateResetPubKeysCommand
func (mr *MockFactoryMockRecorder) CreateResetPubKeysCommand() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateResetPubKeysCommand", reflect.TypeOf((*MockFactory)(nil).CreateResetPubKeysCommand))
}

// CreateResetTopicsCommand mocks base method
func (m *MockFactory) CreateResetTopicsCommand() (Command, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateResetTopicsCommand")
	ret0, _ := ret[0].(Command)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateResetTopicsCommand indicates an expected call of CreateResetTopicsCommand
func (mr *MockFactoryMockRecorder) CreateResetTopicsCommand() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateResetTopicsCommand", reflect.TypeOf((*MockFactory)(nil).CreateResetTopicsCommand))
}

// CreateSetIDKeyCommand mocks base method
func (m *MockFactory) CreateSetIDKeyCommand(arg0 []byte) (Command, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSetIDKeyCommand", arg0)
	ret0, _ := ret[0].(Command)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSetIDKeyCommand indicates an expected call of CreateSetIDKeyCommand
func (mr *MockFactoryMockRecorder) CreateSetIDKeyCommand(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSetIDKeyCommand", reflect.TypeOf((*MockFactory)(nil).CreateSetIDKeyCommand), arg0)
}

// CreateSetPubKeyCommand mocks base method
func (m *MockFactory) CreateSetPubKeyCommand(arg0 ed25519.PublicKey, arg1 string) (Command, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSetPubKeyCommand", arg0, arg1)
	ret0, _ := ret[0].(Command)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSetPubKeyCommand indicates an expected call of CreateSetPubKeyCommand
func (mr *MockFactoryMockRecorder) CreateSetPubKeyCommand(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSetPubKeyCommand", reflect.TypeOf((*MockFactory)(nil).CreateSetPubKeyCommand), arg0, arg1)
}

// CreateSetTopicKeyCommand mocks base method
func (m *MockFactory) CreateSetTopicKeyCommand(arg0, arg1 []byte) (Command, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSetTopicKeyCommand", arg0, arg1)
	ret0, _ := ret[0].(Command)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSetTopicKeyCommand indicates an expected call of CreateSetTopicKeyCommand
func (mr *MockFactoryMockRecorder) CreateSetTopicKeyCommand(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSetTopicKeyCommand", reflect.TypeOf((*MockFactory)(nil).CreateSetTopicKeyCommand), arg0, arg1)
}
