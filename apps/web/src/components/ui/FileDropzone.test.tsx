import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import FileDropzone from './FileDropzone'

function makeFile(name: string, type: string, sizeMB = 1): File {
  const bytes = sizeMB * 1024 * 1024
  return new File([new ArrayBuffer(bytes)], name, { type })
}

function mockFileList(files: File[]): FileList {
  return Object.assign(files, {
    item: (i: number) => files[i] ?? null,
  }) as unknown as FileList
}

function getInput() {
  return document.querySelector('input[type="file"]') as HTMLInputElement
}

function fireFileChange(files: File[]) {
  const input = getInput()
  Object.defineProperty(input, 'files', {
    value: mockFileList(files),
    configurable: true,
  })
  fireEvent.change(input)
}

describe('FileDropzone', () => {
  it('file PDF valid → onFiles dipanggil', () => {
    const onFiles = vi.fn()
    render(<FileDropzone onFiles={onFiles} />)
    const file = makeFile('dokumen.pdf', 'application/pdf', 1)
    fireFileChange([file])
    expect(onFiles).toHaveBeenCalledWith([file])
  })

  it('file bukan PDF → onFiles TIDAK dipanggil, onError dipanggil', () => {
    const onFiles = vi.fn()
    const onError = vi.fn()
    render(<FileDropzone onFiles={onFiles} onError={onError} />)
    const file = makeFile('gambar.png', 'image/png', 1)
    fireFileChange([file])
    expect(onFiles).not.toHaveBeenCalled()
    expect(onError).toHaveBeenCalledOnce()
  })

  it('file melebihi ukuran → onFiles TIDAK dipanggil, onError dipanggil', () => {
    const onFiles = vi.fn()
    const onError = vi.fn()
    render(<FileDropzone onFiles={onFiles} onError={onError} maxSizeMB={2} />)
    const file = makeFile('besar.pdf', 'application/pdf', 5)
    fireFileChange([file])
    expect(onFiles).not.toHaveBeenCalled()
    expect(onError).toHaveBeenCalledOnce()
  })

  it('disabled → zona memiliki aria-disabled', () => {
    render(<FileDropzone onFiles={() => {}} disabled />)
    const zone = screen.getByRole('button')
    expect(zone).toHaveAttribute('aria-disabled')
  })
})
