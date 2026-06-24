import { cn } from '../../lib/cn'
import { scoreColor } from '../../lib/score'

const toneStroke: Record<string, string> = {
  success: 'text-success',
  warning: 'text-warning',
  info:    'text-info',
  danger:  'text-danger',
}

export interface ScoreRingProps {
  score: number
  size?: number
  strokeWidth?: number
  showLabel?: boolean
  className?: string
}

export default function ScoreRing({
  score,
  size = 64,
  strokeWidth = 6,
  showLabel = true,
  className,
}: ScoreRingProps) {
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const clamped = Math.min(100, Math.max(0, score))
  const offset = circumference * (1 - clamped / 100)
  const tone = scoreColor(clamped)
  const strokeColor = toneStroke[tone] ?? 'text-fg-muted'

  const center = size / 2
  const fontSize = size < 40 ? 10 : size < 56 ? 13 : 16

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      role="img"
      aria-label={`Fit score: ${clamped}`}
      className={cn('flex-shrink-0', className)}
    >
      {/* Track */}
      <circle
        cx={center}
        cy={center}
        r={radius}
        fill="none"
        strokeWidth={strokeWidth}
        className="text-surface-subtle stroke-current"
      />
      {/* Progress */}
      <circle
        cx={center}
        cy={center}
        r={radius}
        fill="none"
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        transform={`rotate(-90 ${center} ${center})`}
        className={cn('stroke-current transition-all duration-500', strokeColor)}
      />
      {/* Label */}
      {showLabel && (
        <text
          x={center}
          y={center}
          textAnchor="middle"
          dominantBaseline="central"
          fontSize={fontSize}
          fontWeight="600"
          className="fill-fg tabular-nums font-semibold"
        >
          {clamped}
        </text>
      )}
    </svg>
  )
}
