import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import Modal from './Modal'

function setup(props: Partial<Parameters<typeof Modal>[0]> = {}) {
  const onClose = vi.fn()
  render(
    <Modal
      open={props.open ?? true}
      onClose={props.onClose ?? onClose}
      title={props.title ?? 'Judul Modal'}
      {...props}
    >
      <p>Konten modal</p>
      <button type="button">Tombol Dalam</button>
    </Modal>
  )
  return { onClose }
}

describe('Modal', () => {
  it('tampil saat open=true', () => {
    setup()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Judul Modal')).toBeInTheDocument()
  })

  it('tidak tampil saat open=false', () => {
    setup({ open: false })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('klik tombol X memanggil onClose', () => {
    const { onClose } = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Tutup' }))
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('klik overlay memanggil onClose', () => {
    const { onClose } = setup()
    // overlay adalah div di belakang panel, punya aria-hidden
    const overlay = document.querySelector('.fixed.inset-0 [aria-hidden]') as HTMLElement
    fireEvent.click(overlay)
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('tekan Escape memanggil onClose', () => {
    const { onClose } = setup()
    fireEvent.keyDown(document, { key: 'Escape', bubbles: true })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('modal punya aria-modal dan aria-labelledby', () => {
    setup()
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-labelledby')
  })
})
