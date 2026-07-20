package service

import (
	"strings"
	"testing"
)

func TestNormalizeEmails(t *testing.T) {
	t.Run("merapikan, menurunkan huruf, dan membuang duplikat", func(t *testing.T) {
		got, err := normalizeEmails([]string{"  Budi@Contoh.CO.ID ", "budi@contoh.co.id", "sari@lain.com", ""})
		if err != nil {
			t.Fatalf("tak terduga error: %v", err)
		}
		want := []string{"budi@contoh.co.id", "sari@lain.com"}
		if len(got) != len(want) {
			t.Fatalf("jumlah = %d (%v), want %d", len(got), got, len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("index %d = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("daftar kosong menghasilkan slice kosong, bukan nil", func(t *testing.T) {
		got, err := normalizeEmails(nil)
		if err != nil {
			t.Fatalf("tak terduga error: %v", err)
		}
		if got == nil {
			t.Error("hasil nil — JSON akan mengirim null, bukan []")
		}
		if len(got) != 0 {
			t.Errorf("panjang = %d, want 0", len(got))
		}
	})

	t.Run("menolak alamat yang tidak valid dan menyebut alamatnya", func(t *testing.T) {
		for _, bad := range []string{"bukan-email", "a@b", "a b@c.com", "@contoh.com"} {
			_, err := normalizeEmails([]string{"ok@contoh.com", bad})
			if err == nil {
				t.Errorf("%q lolos padahal tidak valid", bad)
				continue
			}
			// Pesan harus menyebut alamat bermasalah supaya user tahu mana
			// yang perlu dibetulkan dari daftar panjang.
			if !strings.Contains(err.Error(), bad) {
				t.Errorf("pesan error tidak menyebut %q: %v", bad, err)
			}
		}
	})

	t.Run("menolak bentuk 'Nama <email>' yang lolos net/mail", func(t *testing.T) {
		// mail.ParseAddress menerima "Budi <budi@contoh.com>"; kita hanya mau
		// alamat telanjang supaya nilai tersimpan konsisten.
		if _, err := normalizeEmails([]string{"Budi <budi@contoh.com>"}); err == nil {
			t.Error("bentuk dengan nama tampil lolos, seharusnya ditolak")
		}
	})

	t.Run("menolak bila melebihi batas peserta", func(t *testing.T) {
		many := make([]string, 0, maxEventParticipants+1)
		for i := 0; i <= maxEventParticipants; i++ {
			many = append(many, strings.ToLower(string(rune('a'+i%26)))+string(rune('a'+i/26))+"@contoh.com")
		}
		if _, err := normalizeEmails(many); err == nil {
			t.Errorf("%d alamat lolos, batasnya %d", len(many), maxEventParticipants)
		}
	})
}
