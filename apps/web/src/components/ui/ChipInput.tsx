import { useState, useRef } from 'react'
import type { KeyboardEvent } from 'react'
import { X } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface ChipInputProps {
  value: string[]
  onChange: (next: string[]) => void
  presets?: string[]
  placeholder?: string
  disabled?: boolean
  className?: string
}

export default function ChipInput({
  value,
  onChange,
  presets,
  placeholder = 'Tambah…',
  disabled,
  className,
}: ChipInputProps) {
  const [inputVal, setInputVal] = useState('')
  const [focused, setFocused] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  function addChip(raw: string) {
    const chip = raw.trim()
    if (!chip || value.includes(chip)) return
    onChange([...value, chip])
  }

  function removeChip(chip: string) {
    onChange(value.filter((c) => c !== chip))
  }

  function togglePreset(preset: string) {
    if (value.includes(preset)) {
      removeChip(preset)
    } else {
      onChange([...value, preset])
    }
  }

  function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addChip(inputVal)
      setInputVal('')
    } else if (e.key === 'Backspace' && inputVal === '' && value.length > 0) {
      removeChip(value[value.length - 1])
    }
  }

  return (
    <div className={cn('space-y-2', className)}>
      {/* Input area */}
      <div
        onClick={() => !disabled && inputRef.current?.focus()}
        className={cn(
          'flex flex-wrap items-center gap-1.5 min-h-10 px-3 py-2 rounded-btn border border-line bg-surface cursor-text',
          'transition-colors duration-150',
          focused && !disabled && 'ring-2 ring-primary ring-offset-2',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      >
        {value.map((chip) => (
          <span
            key={chip}
            // Neutral slate tags (not emerald): a wall of many free-form tags
            // in the brand color reads as "everything is green". Reserve
            // emerald for genuine accents; tags stay quiet chrome.
            className="inline-flex items-center gap-1 pl-2.5 pr-1.5 py-0.5 rounded-md bg-surface-subtle text-fg border border-line text-caption font-medium"
          >
            {chip}
            {!disabled && (
              <button
                type="button"
                aria-label={`Hapus ${chip}`}
                onClick={(e) => { e.stopPropagation(); removeChip(chip) }}
                className="text-fg-subtle hover:text-danger transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-danger rounded-full"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </span>
        ))}
        <input
          ref={inputRef}
          disabled={disabled}
          value={inputVal}
          placeholder={value.length === 0 ? placeholder : ''}
          onChange={(e) => setInputVal(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => setFocused(true)}
          onBlur={() => {
            setFocused(false)
            if (inputVal.trim()) {
              addChip(inputVal)
              setInputVal('')
            }
          }}
          className="flex-1 min-w-[120px] bg-transparent text-body text-fg placeholder:text-fg-subtle focus:outline-none disabled:cursor-not-allowed"
        />
      </div>

      {/* Preset chips */}
      {presets && presets.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {presets.map((preset) => {
            const selected = value.includes(preset)
            return (
              <button
                key={preset}
                type="button"
                disabled={disabled}
                onClick={() => togglePreset(preset)}
                className={cn(
                  'px-2.5 py-1 rounded-pill text-caption font-medium border transition-colors duration-150',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1',
                  'disabled:opacity-50 disabled:cursor-not-allowed',
                  selected
                    ? 'bg-primary-subtle text-primary-active border-primary-border'
                    : 'bg-surface-subtle text-fg-muted border-line hover:border-primary-border hover:text-fg'
                )}
              >
                {preset}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
