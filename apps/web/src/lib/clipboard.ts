/**
 * Copies text to the clipboard, falling back to the legacy execCommand path
 * when the async Clipboard API is unavailable — e.g. the app served over
 * plain HTTP on a non-localhost origin isn't a "secure context", so
 * `navigator.clipboard` is simply undefined there and calling `.writeText`
 * throws synchronously. Never throws — safe to call directly from onClick
 * handlers (including inside a popover menu item) without try/catch.
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // fall through to legacy fallback below
  }
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}
