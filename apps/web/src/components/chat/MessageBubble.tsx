import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Sparkles } from 'lucide-react'
import { cn } from '../../lib/cn'
import StreamingText from '../ui/StreamingText'
import ToolCallChip from './ToolCallChip'
import type { ToolCall } from '../../api/chat'

export interface MessageBubbleProps {
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: string
  streaming?: boolean
  toolCalls?: ToolCall[]
}

export default function MessageBubble({ role, content, streaming = false, toolCalls }: MessageBubbleProps) {
  if (role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[75%] rounded-card px-4 py-2.5 bg-primary text-white text-body">
          {content}
        </div>
      </div>
    )
  }

  if (role === 'assistant') {
    return (
      <div className="flex justify-start gap-2">
        <div className="mt-1 shrink-0 w-6 h-6 rounded-full bg-accent/10 flex items-center justify-center">
          <Sparkles className="w-3.5 h-3.5 text-accent" aria-hidden="true" />
        </div>
        <div className="max-w-[75%] flex flex-col gap-2">
          {/* Tool call chips (from stored message) */}
          {toolCalls && toolCalls.length > 0 && (
            <div className="flex flex-col gap-1">
              {toolCalls.map((tc) => (
                <ToolCallChip key={tc.id} name={tc.name} arguments={tc.arguments} status="done" />
              ))}
            </div>
          )}

          {/* Message content */}
          {content && (
            <div
              className={cn(
                'rounded-card px-4 py-2.5 text-body text-fg',
                'border border-accent/20 bg-accent/5',
              )}
            >
              {streaming ? (
                <StreamingText text={content} streaming className="text-body" />
              ) : (
                <div className="prose-custom">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    components={{
                      p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                      ul: ({ children }) => <ul className="list-disc pl-4 mb-2 space-y-0.5">{children}</ul>,
                      ol: ({ children }) => <ol className="list-decimal pl-4 mb-2 space-y-0.5">{children}</ol>,
                      li: ({ children }) => <li className="text-body">{children}</li>,
                      strong: ({ children }) => <strong className="font-semibold text-fg">{children}</strong>,
                      em: ({ children }) => <em className="italic">{children}</em>,
                      code: ({ children }) => (
                        <code className="bg-surface-subtle px-1 py-0.5 rounded text-sm font-mono">{children}</code>
                      ),
                      pre: ({ children }) => (
                        <pre className="bg-surface-subtle rounded-card p-3 overflow-x-auto text-sm font-mono mb-2">{children}</pre>
                      ),
                      h1: ({ children }) => <h1 className="text-h3 font-semibold mb-1">{children}</h1>,
                      h2: ({ children }) => <h2 className="text-body font-semibold mb-1">{children}</h2>,
                      h3: ({ children }) => <h3 className="text-body font-medium mb-1">{children}</h3>,
                    }}
                  >
                    {content}
                  </ReactMarkdown>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    )
  }

  // system / tool messages — minimal display
  return (
    <div className="flex justify-center">
      <span className="text-caption text-fg-subtle italic px-3 py-1 bg-surface-subtle rounded-pill border border-line">
        {content || `[${role}]`}
      </span>
    </div>
  )
}
