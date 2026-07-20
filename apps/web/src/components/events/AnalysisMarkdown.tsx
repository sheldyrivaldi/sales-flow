import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

/**
 * Renderer markdown untuk hasil Analisa AI.
 *
 * Analisa yang berguna itu panjang dan bertingkat — bullet, penomoran, tebal,
 * miring, kadang tabel. Menampilkannya sebagai teks polos membuat struktur itu
 * hilang dan hasilnya sulit dipindai. Gaya di sini sengaja mengikuti token
 * aplikasi, bukan gaya bawaan markdown, supaya menyatu dengan halaman.
 */
export default function AnalysisMarkdown({ children }: { children: string }) {
  return (
    <div className="text-body text-fg leading-relaxed">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
          strong: ({ children }) => <strong className="font-semibold text-fg">{children}</strong>,
          em: ({ children }) => <em className="italic">{children}</em>,
          ul: ({ children }) => <ul className="list-disc pl-5 mb-2 last:mb-0 flex flex-col gap-1">{children}</ul>,
          ol: ({ children }) => <ol className="list-decimal pl-5 mb-2 last:mb-0 flex flex-col gap-1">{children}</ol>,
          li: ({ children }) => <li className="pl-0.5">{children}</li>,
          h1: ({ children }) => <h4 className="text-body font-semibold text-fg mt-3 mb-1.5">{children}</h4>,
          h2: ({ children }) => <h4 className="text-body font-semibold text-fg mt-3 mb-1.5">{children}</h4>,
          h3: ({ children }) => <h5 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mt-3 mb-1">{children}</h5>,
          h4: ({ children }) => <h5 className="text-caption font-semibold text-fg-muted uppercase tracking-wide mt-3 mb-1">{children}</h5>,
          code: ({ children }) => (
            <code className="rounded bg-surface-subtle px-1 py-0.5 text-caption font-mono text-fg">{children}</code>
          ),
          pre: ({ children }) => (
            <pre className="rounded-btn bg-surface-subtle p-2.5 overflow-x-auto text-caption mb-2">{children}</pre>
          ),
          blockquote: ({ children }) => (
            <blockquote className="border-l-2 border-primary pl-3 text-fg-muted italic mb-2">{children}</blockquote>
          ),
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
              {children}
            </a>
          ),
          hr: () => <hr className="border-line my-3" />,
          // Tabel dibungkus wrapper ber-scroll sendiri: tabel lebar tidak boleh
          // membuat seluruh halaman ikut bergeser horizontal.
          table: ({ children }) => (
            <div className="overflow-x-auto scrollbar-thin mb-2">
              <table className="w-full text-caption border-collapse">{children}</table>
            </div>
          ),
          thead: ({ children }) => <thead className="border-b border-line">{children}</thead>,
          th: ({ children }) => <th className="text-left py-1.5 pr-3 font-medium text-fg-muted">{children}</th>,
          td: ({ children }) => <td className="py-1.5 pr-3 border-b border-line/60 align-top">{children}</td>,
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
