import { useState, useRef, useId } from 'react'
import { createPortal } from 'react-dom'
import { ChevronDown } from 'lucide-react'
import { cn } from '../../lib/cn'
import { inputBase } from './Input'
import { useFloatingPosition } from '../../lib/useFloatingPosition'

export interface ComboboxOption {
  label: string
  value: string
}

export interface ComboboxProps {
  options: ComboboxOption[]
  value?: string
  onChange?: (value: string) => void
  placeholder?: string
  invalid?: boolean
  disabled?: boolean
  className?: string
}

export default function Combobox({
  options,
  value,
  onChange,
  placeholder = 'Pilih…',
  invalid,
  disabled,
  className,
}: ComboboxProps) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)
  const listboxId = useId()
  const inputRef = useRef<HTMLInputElement>(null)
  const wrapperRef = useRef<HTMLDivElement>(null)
  // Portaled + fixed-positioned (see useFloatingPosition) so the listbox
  // isn't clipped by an ancestor's overflow (e.g. a Modal's scrollable body)
  // the way a plain `absolute` dropdown would be.
  const pos = useFloatingPosition(wrapperRef, open, 'start')

  const selected = options.find((o) => o.value === value)
  const displayQuery = open ? query : (selected?.label ?? '')

  const filtered = options.filter((o) =>
    o.label.toLowerCase().includes(query.toLowerCase())
  )

  function choose(opt: ComboboxOption) {
    onChange?.(opt.value)
    setQuery('')
    setOpen(false)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (!open) {
      if (e.key === 'ArrowDown' || e.key === 'Enter') {
        setOpen(true)
        setActiveIndex(0)
      }
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex((i) => Math.min(i + 1, filtered.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex((i) => Math.max(i - 1, 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (filtered[activeIndex]) choose(filtered[activeIndex])
    } else if (e.key === 'Escape') {
      setOpen(false)
      setQuery('')
    }
  }

  return (
    <div ref={wrapperRef} className="relative">
      <div className="relative">
        <input
          ref={inputRef}
          role="combobox"
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-controls={listboxId}
          aria-activedescendant={open && filtered[activeIndex] ? `${listboxId}-${activeIndex}` : undefined}
          aria-invalid={invalid || undefined}
          disabled={disabled}
          placeholder={placeholder}
          value={displayQuery}
          onFocus={() => {
            setOpen(true)
            setActiveIndex(0)
          }}
          onBlur={() => {
            setTimeout(() => setOpen(false), 150)
          }}
          onChange={(e) => {
            setQuery(e.target.value)
            setOpen(true)
            setActiveIndex(0)
          }}
          onKeyDown={handleKeyDown}
          className={cn(inputBase, 'pr-9', className)}
        />
        <ChevronDown
          className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-fg-muted"
          aria-hidden="true"
        />
      </div>

      {open &&
        filtered.length > 0 &&
        pos &&
        createPortal(
          <ul
            id={listboxId}
            role="listbox"
            style={{
              position: 'fixed',
              top: pos.top,
              left: pos.left,
              width: wrapperRef.current?.getBoundingClientRect().width,
            }}
            className="z-[9000] bg-surface border border-line rounded-btn shadow-lg overflow-auto max-h-52 py-1"
          >
            {filtered.map((opt, i) => (
              <li
                key={opt.value}
                id={`${listboxId}-${i}`}
                role="option"
                aria-selected={opt.value === value}
                onMouseDown={() => choose(opt)}
                className={cn(
                  'px-3 py-2 text-body cursor-pointer',
                  i === activeIndex ? 'bg-surface-subtle text-fg' : 'text-fg hover:bg-surface-subtle'
                )}
              >
                {opt.label}
              </li>
            ))}
          </ul>,
          document.body
        )}
    </div>
  )
}
