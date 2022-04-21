/*
Copyright 2014 The Kubernetes Authors.

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

	"k8s.io/apimachinery/pkg/util/sets"
)

// 测试删除功能 1
func TestThreadSafeStoreDeleteRemovesEmptySetsFromIndex(t *testing.T) {
	testIndexer := "testIndexer"

	indexers := Indexers{
		testIndexer: func(obj interface{}) (strings []string, e error) {
			indexes := []string{obj.(string)}
			return indexes, nil
		},
	}

	indices := Indices{}
	store := NewThreadSafeStore(indexers, indices).(*threadSafeMap)

	testKey := "testKey"

	store.Add(testKey, testKey)

	// Assumption check, there should be a set for the `testKey` with one element in the added index
	set := store.indices[testIndexer][testKey]

	if len(set) != 1 {
		t.Errorf("Initial assumption of index backing string set having 1 element failed. Actual elements: %d", len(set))
		return
	}

	store.Delete(testKey)
	set, present := store.indices[testIndexer][testKey]

	if present {
		t.Errorf("Index backing string set not deleted from index. Set length: %d", len(set))
	}
}

// 测试删除功能 2
func TestThreadSafeStoreAddKeepsNonEmptySetPostDeleteFromIndex(t *testing.T) {
	testIndexer := "testIndexer"
	testIndex := "testIndex"

	indexers := Indexers{
		testIndexer: func(obj interface{}) (strings []string, e error) {
			indexes := []string{testIndex}
			return indexes, nil
		},
	}

	indices := Indices{}
	store := NewThreadSafeStore(indexers, indices).(*threadSafeMap)

	store.Add("retain", "retain")
	store.Add("delete", "delete")

	// Assumption check, there should be a set for the `testIndex` with two elements
	set := store.indices[testIndexer][testIndex]

	if len(set) != 2 {
		t.Errorf("Initial assumption of index backing string set having 2 elements failed. Actual elements: %d", len(set))
		return
	}

	store.Delete("delete")
	set, present := store.indices[testIndexer][testIndex]

	if !present {
		t.Errorf("Index backing string set erroneously deleted from index.")
		return
	}

	if len(set) != 1 {
		t.Errorf("Index backing string set has incorrect length, expect 1. Set length: %d", len(set))
	}
}

// 测试添加索引后的增加逻辑
func TestAddIndexerAfterAdd(t *testing.T) {
	store := NewThreadSafeStore(Indexers{}, Indices{})

	// Add first indexer
	err := store.AddIndexers(Indexers{
		"first": func(obj interface{}) ([]string, error) {
			value := obj.(string)
			return []string{
				value,
			}, nil
		},
	})
	if err != nil {
		t.Errorf("failed to add first indexer")
	}

	// Add some data to index
	store.Add("keya", "value")
	store.Add("keyb", "value")

	// Assert
	// 获取到索引的key值
	indexKeys, _ := store.IndexKeys("first", "value")
	expected := sets.NewString("keya", "keyb")
	actual := sets.NewString(indexKeys...)
	// 校验获取到的值是否一致
	if !actual.Equal(expected) {
		t.Errorf("expected %v does not match actual %v", expected, actual)
	}

	// Add same indexer, which should fail
	err = store.AddIndexers(Indexers{
		"first": func(interface{}) ([]string, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Errorf("Add same index should have failed")
	}

	// Add new indexer
	err = store.AddIndexers(Indexers{
		"second": func(obj interface{}) ([]string, error) {
			v := obj.(string)
			return []string{
				v + "2",
			}, nil
		},
	})
	if err != nil {
		t.Errorf("failed to add second indexer")
	}

	// Assert indexers was added
	if _, ok := store.GetIndexers()["first"]; !ok {
		t.Errorf("missing indexer first")
	}
	if _, ok := store.GetIndexers()["second"]; !ok {
		t.Errorf("missing indexer second")
	}

	// Assert existing data is re-indexed
	indexKeys, _ = store.IndexKeys("first", "value")
	expected = sets.NewString("keya", "keyb")
	actual = sets.NewString(indexKeys...)
	if !actual.Equal(expected) {
		t.Errorf("expected %v does not match actual %v", expected, actual)
	}
	indexKeys, _ = store.IndexKeys("second", "value2")
	expected = sets.NewString("keya", "keyb")
	actual = sets.NewString(indexKeys...)
	if !actual.Equal(expected) {
		t.Errorf("expected %v does not match actual %v", expected, actual)
	}

	// Add more data
	store.Add("keyc", "value")
	store.Add("keyd", "value")

	// Assert new data is indexed
	indexKeys, _ = store.IndexKeys("first", "value")
	expected = sets.NewString("keya", "keyb", "keyc", "keyd")
	actual = sets.NewString(indexKeys...)
	if !actual.Equal(expected) {
		t.Errorf("expected %v does not match actual %v", expected, actual)
	}
	indexKeys, _ = store.IndexKeys("second", "value2")
	expected = sets.NewString("keya", "keyb", "keyc", "keyd")
	actual = sets.NewString(indexKeys...)
	if !actual.Equal(expected) {
		t.Errorf("expected %v does not match actual %v", expected, actual)
	}
}
