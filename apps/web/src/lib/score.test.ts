import { describe, it, expect } from 'vitest'
import { scoreColor, actionColor, toneClasses } from './score'

describe('scoreColor', () => {
  it('0 → danger', () => expect(scoreColor(0)).toBe('danger'))
  it('49 → danger', () => expect(scoreColor(49)).toBe('danger'))
  it('48 → danger (done-check TK-02.4.1)', () => expect(scoreColor(48)).toBe('danger'))
  it('50 → info', () => expect(scoreColor(50)).toBe('info'))
  it('64 → info', () => expect(scoreColor(64)).toBe('info'))
  it('65 → warning', () => expect(scoreColor(65)).toBe('warning'))
  it('79 → warning', () => expect(scoreColor(79)).toBe('warning'))
  it('80 → success', () => expect(scoreColor(80)).toBe('success'))
  it('86 → success (done-check TK-02.4.1)', () => expect(scoreColor(86)).toBe('success'))
  it('100 → success', () => expect(scoreColor(100)).toBe('success'))
})

describe('actionColor', () => {
  it('Pursue → success', () => expect(actionColor('Pursue')).toBe('success'))
  it('Review → warning', () => expect(actionColor('Review')).toBe('warning'))
  it('Watchlist → info', () => expect(actionColor('Watchlist')).toBe('info'))
  it('Reject → danger', () => expect(actionColor('Reject')).toBe('danger'))
  it('Need Partner → accent', () => expect(actionColor('Need Partner')).toBe('accent'))
})

describe('toneClasses', () => {
  it('success → kelas benar', () => {
    expect(toneClasses('success')).toEqual({
      text: 'text-success',
      bg: 'bg-success',
      bgSoft: 'bg-success/10',
      border: 'border-success',
    })
  })
  it('danger → kelas benar', () => {
    expect(toneClasses('danger')).toEqual({
      text: 'text-danger',
      bg: 'bg-danger',
      bgSoft: 'bg-danger/10',
      border: 'border-danger',
    })
  })
})
