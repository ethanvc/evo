// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httpmux

import (
	"net/http"
	"reflect"
	"testing"
)

func TestParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	for i := range ps {
		if val := ps.ByName(ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := ps.ByName("noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func TestHttpMuxInvalidInput(t *testing.T) {
	mux := New[int]()

	recv := catchPanic(func() {
		mux.Handle("", "/", 1)
	})
	if recv == nil {
		t.Fatal("registering empty method did not panic")
	}

	recv = catchPanic(func() {
		mux.GET("", 1)
	})
	if recv == nil {
		t.Fatal("registering empty path did not panic")
	}

	recv = catchPanic(func() {
		mux.GET("noSlashRoot", 1)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}

	recv = catchPanic(func() {
		mux.GET("/", 0)
	})
	if recv != nil {
		t.Fatal("registering zero value should be allowed")
	}

	match, _, _ := mux.Lookup(http.MethodGet, "/")
	if match == nil || match.value != 0 {
		t.Fatalf("want registered zero value, got match=%v", match)
	}
}

func TestHttpMuxGenericLookup(t *testing.T) {
	mux := New[string]()
	mux.Register(http.MethodGet, "/user/:name", "user-list")

	match, params, tsr := mux.Lookup(http.MethodGet, "/user/alice")
	if match == nil {
		t.Fatal("expected matched node")
	}
	if match.value != "user-list" {
		t.Fatalf("want %q, got %q", "user-list", match.value)
	}
	wantParams := Params{{Key: "name", Value: "alice"}}
	if params == nil {
		t.Fatal("expected params")
	}
	if !reflect.DeepEqual(*params, wantParams) {
		t.Fatalf("want params %v, got %v", wantParams, *params)
	}
	PutParams(params)
	if tsr {
		t.Fatal("unexpected tsr")
	}

	match, _, tsr = mux.Lookup(http.MethodGet, "/missing")
	if match != nil {
		t.Fatal("unexpected matched node")
	}
	if tsr {
		t.Fatal("unexpected tsr for missing route")
	}
}

func TestHttpMuxIntLookup(t *testing.T) {
	wantValue := 42
	wantParams := Params{Param{"name", "gopher"}}

	mux := New[int]()

	match, _, tsr := mux.Lookup(http.MethodGet, "/nope")
	if match != nil {
		t.Fatalf("Got node for unregistered pattern: %v", match)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}

	mux.GET("/user/:name", wantValue)
	match, params, _ := mux.Lookup(http.MethodGet, "/user/gopher")
	if match == nil {
		t.Fatal("Got no matched node!")
	}
	if match.value != wantValue {
		t.Fatalf("Got wrong value: want %d, got %d", wantValue, match.value)
	}
	if params == nil {
		t.Fatal("expected params")
	}
	if !reflect.DeepEqual(*params, wantParams) {
		t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, *params)
	}
	PutParams(params)

	mux.GET("/user", wantValue)
	match, params, _ = mux.Lookup(http.MethodGet, "/user")
	if match == nil {
		t.Fatal("Got no matched node!")
	}
	if match.value != wantValue {
		t.Fatalf("Got wrong value: want %d, got %d", wantValue, match.value)
	}
	if params != nil {
		t.Fatalf("Wrong parameter values: want %v, got %v", nil, params)
	}

	match, _, tsr = mux.Lookup(http.MethodGet, "/user/gopher/")
	if match != nil {
		t.Fatalf("Got node for unregistered pattern: %v", match)
	}
	if !tsr {
		t.Error("Got no TSR recommendation!")
	}
}
