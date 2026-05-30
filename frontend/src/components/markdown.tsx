import Markdown from 'react-markdown'
import type { Components } from 'react-markdown'
import { cn } from '@/lib/utils'

const components: Components = {
  h1: ({ children, ...props }) => <h1 className="mt-4 mb-2 text-base font-semibold text-neutral-900 first:mt-0" {...props}>{children}</h1>,
  h2: ({ children, ...props }) => <h2 className="mt-3 mb-1.5 text-sm font-semibold text-neutral-900 first:mt-0" {...props}>{children}</h2>,
  h3: ({ children, ...props }) => <h3 className="mt-3 mb-1 text-sm font-medium text-neutral-800 first:mt-0" {...props}>{children}</h3>,
  p: ({ children, ...props }) => <p className="my-2 text-sm leading-6 text-neutral-700 first:mt-0 last:mb-0" {...props}>{children}</p>,
  ul: ({ children, ...props }) => <ul className="my-2 list-disc space-y-1 pl-5 text-sm text-neutral-700 first:mt-0 last:mb-0" {...props}>{children}</ul>,
  ol: ({ children, ...props }) => <ol className="my-2 list-decimal space-y-1 pl-5 text-sm text-neutral-700 first:mt-0 last:mb-0" {...props}>{children}</ol>,
  li: ({ children, ...props }) => <li className="text-sm leading-6 text-neutral-700" {...props}>{children}</li>,
  code: ({ className, children, ...props }) => {
    const isInline = !className
    return (
      <code
        className={cn(
          isInline
            ? 'rounded bg-neutral-100 px-1 py-0.5 font-mono text-xs text-neutral-800'
            : 'block overflow-auto rounded-lg bg-neutral-100 p-3 font-mono text-xs leading-5 text-neutral-800',
          className as string,
        )}
        {...props}
      >
        {children}
      </code>
    )
  },
  pre: ({ children, ...props }) => <pre className="my-2 first:mt-0 last:mb-0" {...props}>{children}</pre>,
  strong: ({ children, ...props }) => <strong className="font-semibold text-neutral-900" {...props}>{children}</strong>,
  a: ({ children, ...props }) => <a className="text-indigo-500 underline hover:text-indigo-600" target="_blank" rel="noopener noreferrer" {...props}>{children}</a>,
  blockquote: ({ children, ...props }) => <blockquote className="my-2 border-l-2 border-neutral-300 pl-3 text-neutral-600 first:mt-0 last:mb-0" {...props}>{children}</blockquote>,
  hr: (props) => <hr className="my-4 border-neutral-200" {...props} />,
}

export function MarkdownContent({ children, className }: { children: string; className?: string }) {
  return (
    <div className={cn('text-sm', className)}>
      <Markdown components={components}>
        {children}
      </Markdown>
    </div>
  )
}
