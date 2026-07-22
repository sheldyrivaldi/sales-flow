import { navItems } from './navItems'

export interface Crumb {
  label: string
  href?: string
}

// Static trail segments for routes that aren't in navItems at all — entity
// detail pages. Keyed by the path pattern (":id" is a wildcard segment).
// Settings' nested pages (/settings/profile, /settings/ai-agent) don't need
// an entry here — they're resolved from navItems' `children` (childLabel
// below), the same list that drives the sidebar's own nested menu.
const staticLabels: Record<string, string> = {
  '/tenders/:id': 'Detail',
  '/events/:id': 'Detail',
  '/postproject/feedback/new': 'Buat Form',
  '/postproject/feedback/:id': 'Detail',
  '/postproject/feedback/:id/edit': 'Edit',
}

function matchStatic(pathname: string): string | undefined {
  if (staticLabels[pathname]) return staticLabels[pathname]
  const segments = pathname.split('/').filter(Boolean)
  for (const [pattern, label] of Object.entries(staticLabels)) {
    const patternSegments = pattern.split('/').filter(Boolean)
    if (patternSegments.length !== segments.length) continue
    const isMatch = patternSegments.every(
      (seg, i) => seg === segments[i] || seg.startsWith(':')
    )
    if (isMatch) return label
  }
  return undefined
}

function topLevelLabel(path: string): string | undefined {
  return navItems.find((item) => item.path === path)?.label
}

function childLabel(path: string): string | undefined {
  for (const item of navItems) {
    const child = item.children?.find((c) => c.path === path)
    if (child) return child.label
  }
  return undefined
}

/** groupParentOf mengembalikan parent nav yang memiliki `path` sebagai child
 * TANPA berbagi prefix path (grup seperti Pra-Proyek yang anak-anaknya
 * /discovery, /tenders, ... adalah path top-level). Parent berbagi prefix
 * (mis. /settings, /ongoing) sudah tertangani oleh segment-walk biasa. */
function groupParentOf(path: string): string | undefined {
  for (const item of navItems) {
    if (!item.children) continue
    if (path.startsWith(item.path + '/') || path === item.path) continue
    if (item.children.some((c) => c.path === path)) return item.label
  }
  return undefined
}

/** buildTrail maps a pathname to its breadcrumb trail, root-first. Returns a
 * single-item trail (or empty for "/") for top-level pages — callers should
 * only render the breadcrumb bar when the trail has more than one item, since
 * a lone crumb duplicates the page title the Topbar already shows. */
export function buildTrail(pathname: string): Crumb[] {
  if (pathname === '/') return []

  const segments = pathname.split('/').filter(Boolean)
  const trail: Crumb[] = []

  // Walk each ancestor path ("/settings", then "/settings/profile", ...),
  // resolving a label for each from the top-level nav or the static map.
  let acc = ''
  for (let i = 0; i < segments.length; i++) {
    acc += '/' + segments[i]
    const isLast = i === segments.length - 1
    const label = topLevelLabel(acc) ?? childLabel(acc) ?? matchStatic(acc) ?? segments[i]
    trail.push({ label, href: isLast ? undefined : acc })
  }

  // Grup nav tanpa prefix bersama (Pra-Proyek → /tenders dst.): sisipkan
  // crumb parent di depan supaya hirarki menu terbaca — tanpa href karena
  // grup bukan halaman (Back akan jatuh ke "/").
  const groupLabel = groupParentOf('/' + segments[0])
  if (groupLabel) {
    trail.unshift({ label: groupLabel })
  }

  return trail
}
