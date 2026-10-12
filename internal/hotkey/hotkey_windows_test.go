package hotkey

import "testing"

// Registers a combination nobody uses, checks that a second registration of
// it is refused, and that Stop releases it.
func TestListenStop(t *testing.T) {
	binds := []Binding{{Action: "a", Keys: "Ctrl+Alt+Shift+F24"}, {Action: "off", Keys: ""}, {Action: "bad", Keys: "Q"}}
	l, errs := Listen(binds, func(string) {})
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want only the bad binding", errs)
	}

	l2, errs := Listen(binds[:1], func(string) {})
	if len(errs) != 1 {
		t.Errorf("registering a taken hotkey: errs = %v, want 1", errs)
	}
	l2.Stop()

	l.Stop()
	l3, errs := Listen(binds[:1], func(string) {})
	if len(errs) != 0 {
		t.Errorf("hotkey not released by Stop: %v", errs)
	}
	l3.Stop()
}
