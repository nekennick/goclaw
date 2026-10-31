package http

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVaultContentRestoreConfined(t *testing.T) {
	h := &VaultHandler{}
	t.Run("existing and new content can be restored", func(t *testing.T) {
		for _, exists := range []bool{false, true} {
			ws := t.TempDir()
			path := filepath.Join(ws, "note.md")
			if exists {
				if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			undo, err := h.writeDocumentContentWithRestore(ws, "note.md", []byte("replacement"))
			if err != nil || undo == nil {
				t.Fatalf("write: %v", err)
			}
			if err := undo(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if exists {
				if err != nil || string(data) != "original" {
					t.Fatalf("restore: %q %v", data, err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("new file not removed: %v", err)
			}
		}
	})
	if runtime.GOOS == "windows" {
		return
	}
	t.Run("dangling and outside symlinks remain untouched", func(t *testing.T) {
		for _, dangling := range []bool{false, true} {
			ws, outside := t.TempDir(), t.TempDir()
			secret := filepath.Join(outside, "secret.md")
			if !dangling {
				if err := os.WriteFile(secret, []byte("private"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(ws, "note.md")
			if err := os.Symlink(secret, link); err != nil {
				t.Fatal(err)
			}
			undo, err := h.writeDocumentContentWithRestore(ws, "note.md", []byte("replacement"))
			if err == nil || undo != nil {
				t.Fatalf("unsafe target accepted: %v", err)
			}
			if target, err := os.Readlink(link); err != nil || target != secret {
				t.Fatalf("symlink changed: %q %v", target, err)
			}
			if !dangling {
				data, err := os.ReadFile(secret)
				if err != nil || string(data) != "private" {
					t.Fatal("outside bytes changed")
				}
			}
		}
	})
	t.Run("ancestor escape rejected", func(t *testing.T) {
		ws, outside := t.TempDir(), t.TempDir()
		if err := os.Symlink(outside, filepath.Join(ws, "escape")); err != nil {
			t.Fatal(err)
		}
		undo, err := h.writeDocumentContentWithRestore(ws, "escape/note.md", []byte("replacement"))
		if err == nil || undo != nil {
			t.Fatalf("ancestor escape accepted: %v", err)
		}
		if _, err := os.Stat(filepath.Join(outside, "note.md")); !os.IsNotExist(err) {
			t.Fatal("outside file materialized")
		}
	})
}
