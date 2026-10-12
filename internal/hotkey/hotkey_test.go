package hotkey

import "testing"

func TestParse(t *testing.T) {
	ok := []struct {
		in, want string
		mods     uint32
		vk       uint32
	}{
		{"Ctrl+Shift+F1", "Ctrl+Shift+F1", ModCtrl | ModShift, 0x70},
		{"shift + ctrl + f2", "Ctrl+Shift+F2", ModCtrl | ModShift, 0x71},
		{"F9", "F9", 0, 0x78},
		{"Alt+Num5", "Alt+Num5", ModAlt, 0x65},
		{"Win+o", "Win+O", ModWin, 0x4F},
		{"Control+Alt+PageUp", "Ctrl+Alt+PageUp", ModCtrl | ModAlt, 0x21},
		{"Ctrl+Shift+O", "Ctrl+Shift+O", ModCtrl | ModShift, 0x4F},
	}
	for _, tc := range ok {
		c, err := Parse(tc.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.in, err)
			continue
		}
		if c.String() != tc.want || c.Mods != tc.mods || c.VK != tc.vk {
			t.Errorf("Parse(%q) = %s (mods %#x vk %#x), want %s (mods %#x vk %#x)", tc.in, c, c.Mods, c.VK, tc.want, tc.mods, tc.vk)
		}
	}

	for _, in := range []string{"", "Ctrl+", "Ctrl+Ctrl+A", "Hyper+F1", "Ctrl+F25", "A", "Shift+A", "Shift+Space", "F1+Ctrl"} {
		if c, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %s, want error", in, c)
		}
	}
}
