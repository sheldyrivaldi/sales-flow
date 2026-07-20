package repository

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Filter multi-kolom bergantung pada satu hal yang mudah luput: klausa OR untuk
// pencarian teks HARUS dikurung. Tanpa kurung, presedensi SQL membuat
// `type = X AND name ILIKE ? OR organizer ILIKE ?` terbaca sebagai
// `(type = X AND name ILIKE ?) OR (organizer ILIKE ?)` — event bertipe lain
// ikut lolos asal penyelenggaranya cocok.
//
// Diuji sebagai pemeriksaan sumber, bukan lewat database, supaya tetap
// berjalan di CI tanpa Postgres.
func TestEventSearchOrIsParenthesised(t *testing.T) {
	src, err := os.ReadFile("event_repo.go")
	if err != nil {
		t.Fatalf("baca event_repo.go: %v", err)
	}
	text := string(src)

	idx := strings.Index(text, "ILIKE")
	if idx < 0 {
		t.Fatal("tidak menemukan klausa ILIKE — apakah filter pencarian dihapus?")
	}

	// Ambil setiap literal string yang memuat OR + ILIKE dan pastikan dikurung.
	lit := regexp.MustCompile(`"([^"]*ILIKE[^"]*)"`)
	found := 0
	for _, m := range lit.FindAllStringSubmatch(text, -1) {
		clause := m[1]
		if !strings.Contains(clause, " OR ") {
			continue
		}
		found++
		if !strings.HasPrefix(strings.TrimSpace(clause), "(") || !strings.HasSuffix(strings.TrimSpace(clause), ")") {
			t.Errorf("klausa OR tidak dikurung, filter lain bisa bocor:\n  %s", clause)
		}
	}
	if found == 0 {
		t.Error("tidak ada klausa OR pada pencarian — pencarian multi-kolom hilang?")
	}
}

// Pencarian harus menyapu lebih dari satu kolom, sesuai permintaan "multiple
// kolom search".
func TestEventSearchSpansMultipleColumns(t *testing.T) {
	src, err := os.ReadFile("event_repo.go")
	if err != nil {
		t.Fatalf("baca event_repo.go: %v", err)
	}
	text := string(src)
	for _, col := range []string{"name ILIKE", "organizer ILIKE", "location ILIKE", "notes ILIKE"} {
		if !strings.Contains(text, col) {
			t.Errorf("pencarian tidak mencakup kolom: %s", col)
		}
	}
}
