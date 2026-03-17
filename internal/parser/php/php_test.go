package php

import (
	"testing"
)

func TestParseFunctions(t *testing.T) {
	src := []byte(`<?php
function foo() {
    bar();
}

function bar() {
    return 1;
}
`)
	p := New()
	res, err := p.Parse("test.php", src)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range res.Symbols {
		names[s.Name] = true
	}
	if !names["foo"] || !names["bar"] {
		t.Errorf("expected foo and bar symbols, got %v", names)
	}
}

func TestParseCallEdge(t *testing.T) {
	src := []byte(`<?php
function foo() {
    bar();
}
function bar() {}
`)
	p := New()
	res, err := p.Parse("test.php", src)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range res.Calls {
		if c.CallerName == "foo" && c.CalleeName == "bar" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected call edge foo->bar, got %v", res.Calls)
	}
}

func TestParseClassMethod(t *testing.T) {
	src := []byte(`<?php
class MyController {
    public function index() {}
    private function helper() {}
}
`)
	p := New()
	res, err := p.Parse("test.php", src)
	if err != nil {
		t.Fatal(err)
	}
	foundIndex := false
	for _, s := range res.Symbols {
		if s.Name == "index" && s.Kind == "method" {
			foundIndex = true
		}
	}
	if !foundIndex {
		t.Errorf("expected method 'index', got %v", res.Symbols)
	}
}

func TestCallUserFuncCaveat(t *testing.T) {
	src := []byte(`<?php
function foo() {
    call_user_func($callback);
}
`)
	p := New()
	res, err := p.Parse("test.php", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Caveats) == 0 {
		t.Error("expected caveat for call_user_func")
	}
}
