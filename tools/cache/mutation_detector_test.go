//go:build !race
// +build !race

/*
Copyright 2016 The Kubernetes Authors.

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

package cache

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// MutationDetector 监听变化
func TestMutationDetector(t *testing.T) {
	// 获取假的watch
	fakeWatch := watch.NewFake()
	lw := &testLW{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return fakeWatch, nil
		},
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return &v1.PodList{}, nil
		},
	}

	// 制造一个pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "anything",
			Labels: map[string]string{"check": "foo"},
		},
	}

	// 创建stopCh
	stopCh := make(chan struct{})
	defer close(stopCh)
	addReceived := make(chan bool)
	mutationFound := make(chan bool)

	informer := NewSharedInformer(lw, &v1.Pod{}, 1*time.Second).(*sharedIndexInformer)
	informer.cacheMutationDetector = &defaultCacheMutationDetector{
		name:   "name",
		period: 1 * time.Second,
		failureFunc: func(message string) {
			mutationFound <- true
		},
	}
	informer.AddEventHandler(
		ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				addReceived <- true
			},
		},
	)
	go informer.Run(stopCh)

	fakeWatch.Add(pod)

	select {
	case <-addReceived:
	}

	// 这里使用的 MutationDetector 来监听变化的
	pod.Labels["change"] = "true"

	select {
	case <-mutationFound:
	}

}
