// dedupCaseInsensitive removes duplicate strings case-insensitively (trimmed),
// preserving first-seen casing and order. Mirrors the backend helper of the
// same name in internal/service/keyword_service.go so keyword lists produced
// on the client match what the server would store.
export function dedupCaseInsensitive(items: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const raw of items) {
    const s = raw.trim()
    if (!s) continue
    const key = s.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(s)
  }
  return out
}
