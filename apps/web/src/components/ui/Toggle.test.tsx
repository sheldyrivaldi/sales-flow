import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import Toggle from './Toggle'

function setup(checked = false, disabled = false) {
  const onChange = vi.fn()
  render(<Toggle checked={checked} onChange={onChange} disabled={disabled} label="Aktif" />)
  return { onChange, btn: screen.getByRole('switch') }
}

describe('Toggle', () => {
  it('aria-checked mencerminkan prop checked (false)', () => {
    const { btn } = setup(false)
    expect(btn).toHaveAttribute('aria-checked', 'false')
  })

  it('aria-checked mencerminkan prop checked (true)', () => {
    const { btn } = setup(true)
    expect(btn).toHaveAttribute('aria-checked', 'true')
  })

  it('klik → onChange(true) saat checked=false', () => {
    const { onChange, btn } = setup(false)
    fireEvent.click(btn)
    expect(onChange).toHaveBeenCalledWith(true)
  })

  it('klik → onChange(false) saat checked=true', () => {
    const { onChange, btn } = setup(true)
    fireEvent.click(btn)
    expect(onChange).toHaveBeenCalledWith(false)
  })

  it('Space saat fokus → toggle', () => {
    const { onChange, btn } = setup(false)
    fireEvent.keyDown(btn, { key: ' ' })
    expect(onChange).toHaveBeenCalledWith(true)
  })

  it('Enter saat fokus → toggle', () => {
    const { onChange, btn } = setup(false)
    fireEvent.keyDown(btn, { key: 'Enter' })
    expect(onChange).toHaveBeenCalledWith(true)
  })

  it('disabled → klik tidak memanggil onChange', () => {
    const { onChange, btn } = setup(false, true)
    fireEvent.click(btn)
    expect(onChange).not.toHaveBeenCalled()
  })
})
