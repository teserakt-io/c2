// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pb

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt  -destination=api.pb_mocks.go -package pb -self_package github.com/teserakt-io/c2/pkg/pb github.com/teserakt-io/c2/pkg/pb C2_SubscribeToEventStreamClient
