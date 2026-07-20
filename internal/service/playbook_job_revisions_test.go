package service

import (
	"os"
	"strings"
	"testing"
)

// Kolom playbook_job.revisions bersifat NOT NULL, dan slice nil ditulis GORM
// sebagai NULL — INSERT langsung ditolak. Bug ini pernah lolos karena hanya
// jalur custom yang mengisi Revisions, sedangkan jalur event tidak, sehingga
// generate playbook dari halaman detail event selalu gagal 500.
//
// Diuji sebagai pemeriksaan sumber agar tetap jalan tanpa database: setiap
// pembuatan PlaybookJob wajib menyertakan Revisions.
func TestEveryPlaybookJobLiteralInitialisesRevisions(t *testing.T) {
	src := readSource(t, "playbook_job_service.go")

	const marker = "&domain.PlaybookJob{"
	count := 0
	for i := 0; ; {
		idx := strings.Index(src[i:], marker)
		if idx < 0 {
			break
		}
		start := i + idx
		end := strings.Index(src[start:], "\n\t}")
		if end < 0 {
			t.Fatalf("literal PlaybookJob pada offset %d tidak tertutup", start)
		}
		literal := src[start : start+end]
		count++
		if !strings.Contains(literal, "Revisions:") {
			t.Errorf("literal PlaybookJob ke-%d tidak mengisi Revisions — INSERT akan gagal (NOT NULL):\n%s", count, literal)
		}
		i = start + end
	}

	if count == 0 {
		t.Fatal("tidak menemukan satu pun literal &domain.PlaybookJob{}")
	}
	if count < 2 {
		t.Errorf("hanya %d literal ditemukan; diharapkan minimal 2 (custom + event)", count)
	}
}

// readSource membaca berkas sumber di paket ini untuk pemeriksaan statis.
func readSource(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("baca %s: %v", name, err)
	}
	return string(b)
}
