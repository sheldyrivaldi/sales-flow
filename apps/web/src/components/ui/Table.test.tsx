import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import Table from './Table'

interface Row {
  id: string
  nama: string
  nilai: number
}

const data: Row[] = [
  { id: '1', nama: 'Zebra Corp', nilai: 500 },
  { id: '2', nama: 'Alpha PT', nilai: 200 },
  { id: '3', nama: 'Mitra Jaya', nilai: 800 },
]

const columns = [
  { key: 'nama', header: 'Nama', sortable: true },
  { key: 'nilai', header: 'Nilai', sortable: true, align: 'right' as const },
]

function setup(override: Partial<Parameters<typeof Table<Row>>[0]> = {}) {
  render(
    <Table<Row>
      columns={columns}
      data={data}
      rowKey={(r) => r.id}
      pageSize={2}
      {...override}
    />
  )
}

describe('Table — sort', () => {
  it('klik header sortable → baris terurut asc', () => {
    setup()
    fireEvent.click(screen.getByRole('columnheader', { name: /Nama/ }))
    const cells = screen.getAllByRole('cell').filter((_, i) => i % 2 === 0) // kolom nama saja
    expect(cells[0].textContent).toBe('Alpha PT')
  })

  it('klik dua kali header → urutan desc', () => {
    setup()
    const header = screen.getByRole('columnheader', { name: /Nama/ })
    fireEvent.click(header)
    fireEvent.click(header)
    const cells = screen.getAllByRole('cell').filter((_, i) => i % 2 === 0)
    expect(cells[0].textContent).toBe('Zebra Corp')
  })
})

describe('Table — pagination', () => {
  it('halaman 1 tampilkan pageSize baris', () => {
    setup()
    // Dengan pageSize=2, halaman 1 menampilkan 2 baris
    const rows = screen.getAllByRole('row').slice(1) // hapus header row
    expect(rows.length).toBe(2)
  })

  it('klik Berikutnya → halaman 2', () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    // halaman 2: baris ke-3
    expect(screen.getByText('Mitra Jaya')).toBeInTheDocument()
  })

  it('halaman pertama menonaktifkan tombol Sebelumnya', () => {
    setup()
    expect(screen.getByRole('button', { name: 'Sebelumnya' })).toBeDisabled()
  })
})

describe('Table — kebab', () => {
  it('klik MoreVertical → menu muncul', () => {
    const onClick = vi.fn()
    setup({ kebabActions: () => [{ label: 'Hapus', onClick, danger: true }] })
    const triggerBtns = screen.getAllByRole('button', { name: 'Opsi' })
    fireEvent.click(triggerBtns[0])
    expect(screen.getByRole('menuitem', { name: 'Hapus' })).toBeInTheDocument()
  })

  it('klik aksi kebab → callback dipanggil & menu tutup', () => {
    const onClick = vi.fn()
    setup({ kebabActions: () => [{ label: 'Edit', onClick }] })
    fireEvent.click(screen.getAllByRole('button', { name: 'Opsi' })[0])
    fireEvent.click(screen.getByRole('menuitem', { name: 'Edit' }))
    expect(onClick).toHaveBeenCalledOnce()
    expect(screen.queryByRole('menuitem')).not.toBeInTheDocument()
  })
})
