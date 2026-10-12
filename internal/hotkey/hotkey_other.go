//go:build !windows

package hotkey

// Listener is a no-op outside Windows.
type Listener struct{}

func Listen(binds []Binding, on func(action string)) (*Listener, []error) {
	for _, b := range binds {
		if b.Keys != "" {
			return &Listener{}, []error{ErrUnsupported}
		}
	}
	return &Listener{}, nil
}

func (l *Listener) Stop() {}
