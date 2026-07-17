import { clsx, type ClassValue } from 'clsx'
import { extendTailwindMerge } from 'tailwind-merge'

// Default tailwind-merge doesn't know our custom font-size tokens
// (text-h1/h2/h3/body/caption, from tokens.css) and misclassifies them as
// text-color utilities — the same conflict group as text-white/text-<tone>.
// Without this, cn(`text-white`, `text-body`) drops text-white (last one in
// the "color" group wins), leaving text with no color at all.
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      'font-size': [{ text: ['h1', 'h2', 'h3', 'body', 'caption'] }],
    },
  },
})

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
