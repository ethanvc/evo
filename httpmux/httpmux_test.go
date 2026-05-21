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
	mux := New[Handler]()
	handler := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	recv := catchPanic(func() {
		mux.Handle("", "/", handler)
	})
	if recv == nil {
		t.Fatal("registering empty method did not panic")
	}

	recv = catchPanic(func() {
		mux.GET("", handler)
	})
	if recv == nil {
		t.Fatal("registering empty path did not panic")
	}

	recv = catchPanic(func() {
		mux.GET("noSlashRoot", handler)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}

	recv = catchPanic(func() {
		mux.GET("/", nil)
	})
	if recv == nil {
		t.Fatal("registering nil handler did not panic")
	}
}

func TestHttpMuxGenericLookup(t *testing.T) {
	mux := New[string]()
	mux.Register(http.MethodGet, "/user/:name", "user-list")

	value, params, tsr := mux.Lookup(http.MethodGet, "/user/alice")
	if value != "user-list" {
		t.Fatalf("want %q, got %q", "user-list", value)
	}
	wantParams := Params{{Key: "name", Value: "alice"}}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("want params %v, got %v", wantParams, params)
	}
	if tsr {
		t.Fatal("unexpected tsr")
	}

	_, _, tsr = mux.Lookup(http.MethodGet, "/missing")
	if tsr {
		t.Fatal("unexpected tsr for missing route")
	}
}

func TestHttpMuxHandlerLookup(t *testing.T) {
	routed := false
	wantHandler := func(_ http.ResponseWriter, _ *http.Request, _ Params) {
		routed = true
	}
	wantParams := Params{Param{"name", "gopher"}}

	mux := New[Handler]()

	handle, _, tsr := mux.Lookup(http.MethodGet, "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}

	mux.GET("/user/:name", wantHandler)
	handle, params, _ := mux.Lookup(http.MethodGet, "/user/gopher")
	if handle == nil {
		t.Fatal("Got no handle!")
	}
	handle(nil, nil, nil)
	if !routed {
		t.Fatal("Routing failed!")
	}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, params)
	}

	mux.GET("/user", wantHandler)
	handle, params, _ = mux.Lookup(http.MethodGet, "/user")
	if handle == nil {
		t.Fatal("Got no handle!")
	}
	if params != nil {
		t.Fatalf("Wrong parameter values: want %v, got %v", nil, params)
	}

	handle, _, tsr = mux.Lookup(http.MethodGet, "/user/gopher/")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if !tsr {
		t.Error("Got no TSR recommendation!")
	}
}