import { describe, it, expect } from 'vitest'
import { sanitizeSvg, checkSvgGeometry, prepareAiSvg } from './playbookSvg'

const wrap = (inner: string) =>
  `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 1280 720'>${inner}</svg>`

describe('sanitizeSvg', () => {
  it('membuang <script>', () => {
    const out = sanitizeSvg(wrap(`<script>alert(1)</script><rect x='10' y='10' width='5' height='5'/>`))
    expect(out).not.toBeNull()
    expect(out!.toLowerCase()).not.toContain('<script')
    expect(out).toContain('rect')
  })

  it('membuang foreignObject yang bisa menyisipkan HTML', () => {
    const out = sanitizeSvg(wrap(`<foreignObject><div>halo</div></foreignObject>`))
    expect(out!.toLowerCase()).not.toContain('foreignobject')
  })

  it('membuang handler event on*', () => {
    const out = sanitizeSvg(wrap(`<rect x='1' y='1' width='2' height='2' onclick='steal()' onload='x()'/>`))
    expect(out!.toLowerCase()).not.toContain('onclick')
    expect(out!.toLowerCase()).not.toContain('onload')
  })

  it('membuang rujukan eksternal tapi mempertahankan rujukan internal', () => {
    const ext = sanitizeSvg(wrap(`<use href='https://jahat.example/x.svg'/>`))
    expect(ext).not.toContain('jahat.example')
    const int = sanitizeSvg(wrap(`<use href='#ikon'/>`))
    expect(int).toContain('#ikon')
  })

  it('membuang fill url() eksternal tapi mempertahankan gradient internal', () => {
    const ext = sanitizeSvg(wrap(`<rect x='1' y='1' width='2' height='2' fill='url(https://jahat.example/a)'/>`))
    expect(ext).not.toContain('jahat.example')
    const int = sanitizeSvg(wrap(`<rect x='1' y='1' width='2' height='2' fill='url(#grad)'/>`))
    expect(int).toContain('url(#grad)')
  })

  it('memaksa viewBox ke ukuran deck', () => {
    const out = sanitizeSvg(`<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 50'><rect x='1' y='1' width='2' height='2'/></svg>`)
    expect(out).toContain('viewBox="0 0 1280 720"')
  })

  it('menyelamatkan slide yang punya "&" telanjang di teks', () => {
    // Kasus nyata dari model: "Operasi & Pemeliharaan" menggugurkan seluruh
    // slide karena XML menganggap "&" awal entity reference.
    const out = sanitizeSvg(wrap(`<text x='64' y='100' font-size='20'>Operasi &amp; Pemeliharaan</text>`).replace('&amp;', '&'))
    expect(out).not.toBeNull()
    expect(out).toContain('Operasi')
    expect(out).toContain('Pemeliharaan')
  })

  it('tidak merusak entity yang sudah benar', () => {
    const out = sanitizeSvg(wrap(`<text x='64' y='100' font-size='20'>A &amp; B &#39;C&#39; &lt;D&gt;</text>`))
    expect(out).not.toBeNull()
    expect(out).not.toContain('&amp;amp;')
  })

  it('menolak markup yang bukan SVG', () => {
    expect(sanitizeSvg('<html><body>bukan svg</body></html>')).toBeNull()
    expect(sanitizeSvg('bukan markup sama sekali')).toBeNull()
    expect(sanitizeSvg('')).toBeNull()
  })

  it('mempertahankan elemen desain yang sah', () => {
    const out = sanitizeSvg(
      wrap(
        `<defs><linearGradient id='g'><stop offset='0' stop-color='#059669'/></linearGradient></defs>` +
          `<path d='M 0 0 L 10 10' stroke='#fff' stroke-width='2'/>` +
          `<text x='64' y='100' font-family='Arial' font-size='20' fill='#0f172a'>Halo</text>`,
      ),
    )
    expect(out).toContain('linearGradient')
    expect(out).toContain('path')
    expect(out).toContain('Halo')
  })
})

describe('checkSvgGeometry', () => {
  it('menerima teks yang berada di dalam kanvas', () => {
    const r = checkSvgGeometry(wrap(`<text x='64' y='100' font-size='20'>Aman</text>`))
    expect(r.ok).toBe(true)
  })

  it('menolak teks yang keluar kanvas', () => {
    const r = checkSvgGeometry(wrap(`<text x='1500' y='100' font-size='20'>Meluber</text>`))
    expect(r.ok).toBe(false)
    expect(r.reasons.join(' ')).toContain('keluar kanvas')
  })

  it('menolak teks yang tumpang tindih', () => {
    const r = checkSvgGeometry(
      wrap(
        `<text x='100' y='200' font-size='28'>Judul yang panjang sekali</text>` +
          `<text x='100' y='205' font-size='28'>Menimpa judul di atas</text>`,
      ),
    )
    expect(r.ok).toBe(false)
    expect(r.reasons.join(' ')).toContain('tumpang tindih')
  })
})

describe('koordinat bertransform (regresi)', () => {
  // Bug nyata: getBBox mengembalikan koordinat LOKAL, sehingga teks di dalam
  // <g transform='translate(...)'> dibandingkan dengan teks di luar grup dan
  // dianggap bertabrakan. Akibatnya 5 dari 9 slide bagus ikut dibuang.
  // Slide di bawah ini RAPI: judul di atas, konten digeser jauh ke bawah.
  it('tidak menolak slide rapi yang memakai transform', () => {
    const svg = wrap(
      `<text x='64' y='100' font-size='34'>Judul Slide</text>` +
        `<g transform='translate(64,300)'>` +
        `<text x='0' y='0' font-size='20'>Konten pertama</text>` +
        `<text x='0' y='60' font-size='20'>Konten kedua</text>` +
        `</g>`,
    )
    expect(checkSvgGeometry(svg).ok).toBe(true)
  })

  it('tidak menolak konten bertransform yang koordinat lokalnya negatif', () => {
    // y lokal negatif itu SAH selama grupnya digeser ke bawah.
    const svg = wrap(
      `<text x='64' y='100' font-size='34'>Judul</text>` +
        `<g transform='translate(64,400)'><text x='0' y='-20' font-size='18'>Kondisi sekarang</text></g>`,
    )
    expect(checkSvgGeometry(svg).ok).toBe(true)
  })
})

describe('prepareAiSvg', () => {
  it('mengembalikan null agar renderer jatuh ke katalog saat slide cacat', () => {
    expect(prepareAiSvg(undefined)).toBeNull()
    expect(prepareAiSvg('bukan svg')).toBeNull()
    expect(prepareAiSvg(wrap(`<text x='2000' y='100' font-size='20'>Meluber</text>`))).toBeNull()
  })

  it('meloloskan slide yang bersih dan rapi', () => {
    const ok = prepareAiSvg(
      wrap(`<rect width='1280' height='720' fill='#04231b'/><text x='64' y='120' font-size='40' fill='#fff'>Judul</text>`),
    )
    expect(ok).not.toBeNull()
    expect(ok).toContain('Judul')
  })
})
