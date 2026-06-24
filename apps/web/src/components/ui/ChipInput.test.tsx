import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import ChipInput from './ChipInput'

function setup(props: Partial<Parameters<typeof ChipInput>[0]> = {}) {
  const onChange = vi.fn()
  render(
    <ChipInput
      value={props.value ?? []}
      onChange={props.onChange ?? onChange}
      presets={props.presets}
      disabled={props.disabled}
    />
  )
  return { onChange, input: screen.getByRole('textbox') }
}

describe('ChipInput — free add', () => {
  it('Enter → chip muncul & onChange dipanggil', () => {
    const { onChange, input } = setup()
    fireEvent.change(input, { target: { value: 'AI' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onChange).toHaveBeenCalledWith(['AI'])
  })

  it('koma → chip muncul & onChange dipanggil', () => {
    const { onChange, input } = setup()
    fireEvent.change(input, { target: { value: 'Konstruksi' } })
    fireEvent.keyDown(input, { key: ',' })
    expect(onChange).toHaveBeenCalledWith(['Konstruksi'])
  })

  it('duplikat → diabaikan (onChange tidak dipanggil)', () => {
    const { onChange, input } = setup({ value: ['AI'] })
    fireEvent.change(input, { target: { value: 'AI' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onChange).not.toHaveBeenCalled()
  })

  it('input kosong → tidak menambah chip', () => {
    const { onChange, input } = setup()
    fireEvent.change(input, { target: { value: '   ' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onChange).not.toHaveBeenCalled()
  })
})

describe('ChipInput — remove', () => {
  it('klik tombol remove chip → onChange tanpa chip itu', () => {
    const onChange = vi.fn()
    render(<ChipInput value={['AI', 'IT']} onChange={onChange} />)
    const removeBtn = screen.getByRole('button', { name: 'Hapus AI' })
    fireEvent.click(removeBtn)
    expect(onChange).toHaveBeenCalledWith(['IT'])
  })

  it('Backspace saat input kosong → hapus chip terakhir', () => {
    const onChange = vi.fn()
    render(<ChipInput value={['AI', 'IT']} onChange={onChange} />)
    const input = screen.getByRole('textbox')
    fireEvent.keyDown(input, { key: 'Backspace' })
    expect(onChange).toHaveBeenCalledWith(['AI'])
  })
})

describe('ChipInput — preset', () => {
  it('klik preset yang belum aktif → tambah ke value', () => {
    const onChange = vi.fn()
    render(<ChipInput value={[]} onChange={onChange} presets={['IT', 'Konstruksi']} />)
    fireEvent.click(screen.getByRole('button', { name: 'IT' }))
    expect(onChange).toHaveBeenCalledWith(['IT'])
  })

  it('klik preset yang sudah aktif → hapus dari value', () => {
    const onChange = vi.fn()
    render(<ChipInput value={['IT']} onChange={onChange} presets={['IT', 'Konstruksi']} />)
    fireEvent.click(screen.getByRole('button', { name: 'IT' }))
    expect(onChange).toHaveBeenCalledWith([])
  })
})
