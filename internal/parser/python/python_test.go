package python

import (
	"testing"
)

func TestParseFunctions(t *testing.T) {
	src := []byte(`
def foo():
    bar()

def bar():
    pass
`)
	p := New()
	res, err := p.Parse("test.py", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(res.Symbols))
	}
	names := map[string]bool{}
	for _, s := range res.Symbols {
		names[s.Name] = true
	}
	if !names["foo"] || !names["bar"] {
		t.Errorf("expected foo and bar, got %v", names)
	}
}

func TestParseCallEdge(t *testing.T) {
	src := []byte(`
def foo():
    bar()

def bar():
    pass
`)
	p := New()
	res, err := p.Parse("test.py", src)
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

func TestParseTestEntryPoint(t *testing.T) {
	src := []byte(`
def test_something():
    pass
`)
	p := New()
	res, err := p.Parse("test.py", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) == 0 {
		t.Fatal("expected at least one symbol")
	}
	if !res.Symbols[0].IsEntryPoint {
		t.Errorf("expected test_ function to be entry point")
	}
}

func TestParseClassMethod(t *testing.T) {
	src := []byte(`
class MyClass:
    def my_method(self):
        pass
`)
	p := New()
	res, err := p.Parse("test.py", src)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range res.Symbols {
		if s.Name == "my_method" && s.Kind == "method" && s.ParentName == "MyClass" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected method my_method with parent MyClass, got %v", res.Symbols)
	}
}

func TestGetattrCaveat(t *testing.T) {
	src := []byte(`
def foo():
    getattr(obj, "method")()
`)
	p := New()
	res, err := p.Parse("test.py", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Caveats) == 0 {
		t.Error("expected caveat for getattr usage")
	}
}
