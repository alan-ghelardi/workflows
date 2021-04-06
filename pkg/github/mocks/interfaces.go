// /*
// Copyright 2021 The Workflows Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
//

// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/github/interfaces.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	github "github.com/google/go-github/v33/github"
)

// MockcontentsService is a mock of contentsService interface.
type MockcontentsService struct {
	ctrl     *gomock.Controller
	recorder *MockcontentsServiceMockRecorder
}

// MockcontentsServiceMockRecorder is the mock recorder for MockcontentsService.
type MockcontentsServiceMockRecorder struct {
	mock *MockcontentsService
}

// NewMockcontentsService creates a new mock instance.
func NewMockcontentsService(ctrl *gomock.Controller) *MockcontentsService {
	mock := &MockcontentsService{ctrl: ctrl}
	mock.recorder = &MockcontentsServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockcontentsService) EXPECT() *MockcontentsServiceMockRecorder {
	return m.recorder
}

// GetContents mocks base method.
func (m *MockcontentsService) GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContents", ctx, owner, repo, path, opts)
	ret0, _ := ret[0].(*github.RepositoryContent)
	ret1, _ := ret[1].([]*github.RepositoryContent)
	ret2, _ := ret[2].(*github.Response)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// GetContents indicates an expected call of GetContents.
func (mr *MockcontentsServiceMockRecorder) GetContents(ctx, owner, repo, path, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContents", reflect.TypeOf((*MockcontentsService)(nil).GetContents), ctx, owner, repo, path, opts)
}

// MockkeysService is a mock of keysService interface.
type MockkeysService struct {
	ctrl     *gomock.Controller
	recorder *MockkeysServiceMockRecorder
}

// MockkeysServiceMockRecorder is the mock recorder for MockkeysService.
type MockkeysServiceMockRecorder struct {
	mock *MockkeysService
}

// NewMockkeysService creates a new mock instance.
func NewMockkeysService(ctrl *gomock.Controller) *MockkeysService {
	mock := &MockkeysService{ctrl: ctrl}
	mock.recorder = &MockkeysServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockkeysService) EXPECT() *MockkeysServiceMockRecorder {
	return m.recorder
}

// CreateKey mocks base method.
func (m *MockkeysService) CreateKey(ctx context.Context, owner, repo string, key *github.Key) (*github.Key, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateKey", ctx, owner, repo, key)
	ret0, _ := ret[0].(*github.Key)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CreateKey indicates an expected call of CreateKey.
func (mr *MockkeysServiceMockRecorder) CreateKey(ctx, owner, repo, key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateKey", reflect.TypeOf((*MockkeysService)(nil).CreateKey), ctx, owner, repo, key)
}

// DeleteKey mocks base method.
func (m *MockkeysService) DeleteKey(ctx context.Context, owner, repo string, id int64) (*github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteKey", ctx, owner, repo, id)
	ret0, _ := ret[0].(*github.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteKey indicates an expected call of DeleteKey.
func (mr *MockkeysServiceMockRecorder) DeleteKey(ctx, owner, repo, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteKey", reflect.TypeOf((*MockkeysService)(nil).DeleteKey), ctx, owner, repo, id)
}

// GetKey mocks base method.
func (m *MockkeysService) GetKey(ctx context.Context, owner, repo string, id int64) (*github.Key, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKey", ctx, owner, repo, id)
	ret0, _ := ret[0].(*github.Key)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetKey indicates an expected call of GetKey.
func (mr *MockkeysServiceMockRecorder) GetKey(ctx, owner, repo, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKey", reflect.TypeOf((*MockkeysService)(nil).GetKey), ctx, owner, repo, id)
}

// MockhooksService is a mock of hooksService interface.
type MockhooksService struct {
	ctrl     *gomock.Controller
	recorder *MockhooksServiceMockRecorder
}

// MockhooksServiceMockRecorder is the mock recorder for MockhooksService.
type MockhooksServiceMockRecorder struct {
	mock *MockhooksService
}

// NewMockhooksService creates a new mock instance.
func NewMockhooksService(ctrl *gomock.Controller) *MockhooksService {
	mock := &MockhooksService{ctrl: ctrl}
	mock.recorder = &MockhooksServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockhooksService) EXPECT() *MockhooksServiceMockRecorder {
	return m.recorder
}

// CreateHook mocks base method.
func (m *MockhooksService) CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateHook", ctx, owner, repo, hook)
	ret0, _ := ret[0].(*github.Hook)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CreateHook indicates an expected call of CreateHook.
func (mr *MockhooksServiceMockRecorder) CreateHook(ctx, owner, repo, hook interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateHook", reflect.TypeOf((*MockhooksService)(nil).CreateHook), ctx, owner, repo, hook)
}

// DeleteHook mocks base method.
func (m *MockhooksService) DeleteHook(ctx context.Context, owner, repo string, id int64) (*github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteHook", ctx, owner, repo, id)
	ret0, _ := ret[0].(*github.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteHook indicates an expected call of DeleteHook.
func (mr *MockhooksServiceMockRecorder) DeleteHook(ctx, owner, repo, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteHook", reflect.TypeOf((*MockhooksService)(nil).DeleteHook), ctx, owner, repo, id)
}

// EditHook mocks base method.
func (m *MockhooksService) EditHook(ctx context.Context, owner, repo string, id int64, hook *github.Hook) (*github.Hook, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EditHook", ctx, owner, repo, id, hook)
	ret0, _ := ret[0].(*github.Hook)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// EditHook indicates an expected call of EditHook.
func (mr *MockhooksServiceMockRecorder) EditHook(ctx, owner, repo, id, hook interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EditHook", reflect.TypeOf((*MockhooksService)(nil).EditHook), ctx, owner, repo, id, hook)
}

// GetHook mocks base method.
func (m *MockhooksService) GetHook(ctx context.Context, owner, repo string, id int64) (*github.Hook, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHook", ctx, owner, repo, id)
	ret0, _ := ret[0].(*github.Hook)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetHook indicates an expected call of GetHook.
func (mr *MockhooksServiceMockRecorder) GetHook(ctx, owner, repo, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHook", reflect.TypeOf((*MockhooksService)(nil).GetHook), ctx, owner, repo, id)
}

// MockrepositoriesService is a mock of repositoriesService interface.
type MockrepositoriesService struct {
	ctrl     *gomock.Controller
	recorder *MockrepositoriesServiceMockRecorder
}

// MockrepositoriesServiceMockRecorder is the mock recorder for MockrepositoriesService.
type MockrepositoriesServiceMockRecorder struct {
	mock *MockrepositoriesService
}

// NewMockrepositoriesService creates a new mock instance.
func NewMockrepositoriesService(ctrl *gomock.Controller) *MockrepositoriesService {
	mock := &MockrepositoriesService{ctrl: ctrl}
	mock.recorder = &MockrepositoriesServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockrepositoriesService) EXPECT() *MockrepositoriesServiceMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockrepositoriesService) Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, owner, repo)
	ret0, _ := ret[0].(*github.Repository)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockrepositoriesServiceMockRecorder) Get(ctx, owner, repo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockrepositoriesService)(nil).Get), ctx, owner, repo)
}
