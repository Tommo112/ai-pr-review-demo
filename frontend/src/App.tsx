import { useState, useEffect } from 'react'
import type { FormEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { MarkdownContent } from '@/components/markdown'
import { ErrorBoundary } from '@/components/error-boundary'
import { useTypewriter } from '@/lib/use-typewriter'
import { cn } from '@/lib/utils'
import {
  Sparkles, FileCode, AlertTriangle, Copy, Check, Loader2,
  Plus, Minus, PanelLeftClose, PanelLeft, X,
  ChevronRight, MessageSquare, FileText,
} from 'lucide-react'

/* ---------- types ---------- */

type PullRequestFile = {
  filename: string
  status: string
  additions: number
  deletions: number
  patch: string
}

type Risk = {
  level: string
  file: string
  line: number
  title: string
  description: string
  suggestion: string
}

type ReviewComment = {
  file: string
  line: number
  comment: string
}

type PRInfo = {
  title: string
  author: string
  files_changed: number
  additions: number
  deletions: number
}

type ReviewResponse = {
  pr: PRInfo
  files: PullRequestFile[]
  summary: string
  risks: Risk[]
  review_comments: ReviewComment[]
  final_review: string
}

/* ---------- helpers ---------- */

type AppError = { message: string; code?: string }

type ServiceStatus = {
  port: string
  github: { token_configured: boolean }
  ai: {
    enabled: boolean
    api_key_configured: boolean
    model_configured: boolean
    base_url: string
  }
}

function parseSSEBlock(block: string) {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) { event = line.slice('event:'.length).trim(); continue }
    if (line.startsWith('data:')) dataLines.push(line.slice('data:'.length).trimStart())
  }
  return { event, data: dataLines.join('\n') }
}

function isAppError(err: unknown): err is AppError {
  return typeof err === 'object' && err !== null && 'message' in err
}

/* ---------- App ---------- */

function App() {
  const [prUrl, setPrUrl] = useState('https://github.com/gin-gonic/gin/pull/3950')
  const [review, setReview] = useState<ReviewResponse | null>(null)
  const [selectedFile, setSelectedFile] = useState('')
  const [error, setError] = useState('')
  const [copied, setCopied] = useState(false)
  const [serviceStatus, setServiceStatus] = useState<ServiceStatus | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [statusMessage, setStatusMessage] = useState('')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [tab, setTab] = useState<'summary' | 'risks' | 'comments' | 'final'>('summary')

  const hasResults = review !== null

  useEffect(() => {
    fetch('/api/status')
      .then((res) => res.json())
      .then((data) => setServiceStatus(data as ServiceStatus))
      .catch(() => setServiceStatus(null))
  }, [])

  /* ---- data fetching (same logic, new SSE events) ---- */

  async function analyzePR(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsLoading(true)
    setStatusMessage('')
    setError('')
    setReview(null)
    setSelectedFile('')
    setCopied(false)
    setSidebarOpen(true)
    setTab('summary')

    try {
      const response = await fetch('/api/review/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pr_url: prUrl }),
      })
      if (!response.ok) {
        const errBody = (await response.json().catch(() => null)) as { error?: string; code?: string } | null
        throw { message: errBody?.error ?? `Request failed with ${response.status}`, code: errBody?.code }
      }
      if (!response.body) throw new Error('Streaming response is not available')

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const blocks = buffer.split('\n\n')
        buffer = blocks.pop() ?? ''

        for (const block of blocks) {
          const { event, data } = parseSSEBlock(block)
          if (data === '') continue

          try {
            if (event === 'status') {
              const s = JSON.parse(data) as { message: string }
              if (s.message === 'fetching_pr') setStatusMessage('Fetching PR from GitHub...')
              else if (s.message === 'analyzing_ai') setStatusMessage('AI is analyzing the code...')
              continue
            }
            if (event === 'pr') {
              const parsed = JSON.parse(data) as { pr: PRInfo; files: PullRequestFile[] }
              setReview({ pr: parsed.pr, files: parsed.files, summary: '', risks: [], review_comments: [], final_review: '' })
              continue
            }
            if (event === 'summary_delta') {
              const { text } = JSON.parse(data) as { text: string }
              setReview((prev) => (prev ? { ...prev, summary: prev.summary + text } : prev))
              continue
            }
            if (event === 'risk') {
              const { risk } = JSON.parse(data) as { risk: Risk }
              setReview((prev) => (prev ? { ...prev, risks: [...prev.risks, risk] } : prev))
              continue
            }
            if (event === 'review_comment') {
              const { comment } = JSON.parse(data) as { comment: ReviewComment }
              setReview((prev) => (prev ? { ...prev, review_comments: [...prev.review_comments, comment] } : prev))
              continue
            }
            if (event === 'final_review_delta') {
              const { text } = JSON.parse(data) as { text: string }
              setReview((prev) => (prev ? { ...prev, final_review: prev.final_review + text } : prev))
              continue
            }
            if (event === 'review') {
              const doneData = JSON.parse(data) as { review: ReviewResponse }
              setReview((prev) => {
                if (!prev) return doneData.review
                return {
                  ...prev,
                  summary: doneData.review.summary || prev.summary,
                  risks: doneData.review.risks?.length ? doneData.review.risks : prev.risks,
                  review_comments: doneData.review.review_comments?.length ? doneData.review.review_comments : prev.review_comments,
                  final_review: doneData.review.final_review || prev.final_review,
                }
              })
              continue
            }
            if (event === 'error') {
              const errData = JSON.parse(data) as { error: string; code?: string }
              throw { message: errData.error, code: errData.code }
            }
          } catch {
            // skip malformed SSE event, keep streaming
          }
        }
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message
        : isAppError(err) ? err.message
        : 'Request failed'
      const code = isAppError(err) ? err.code : undefined
      setError(code ? `[${code}] ${msg}` : msg)
    } finally {
      setIsLoading(false)
    }
  }

  async function copyFinalReview() {
    if (!review?.final_review) return
    try {
      await navigator.clipboard.writeText(review.final_review)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch { /* ignore */ }
  }

  /* ---- derived ---- */

  const activeFile = review?.files?.find((f) => f.filename === selectedFile)
  const patchLines = activeFile?.patch?.split('\n') ?? []
  const diffOpen = selectedFile !== '' && activeFile !== undefined

  function toggleFile(filename: string) {
    setSelectedFile((prev) => (prev === filename ? '' : filename))
  }

  const typedSummary = useTypewriter(review?.summary ?? '', isLoading)
  const typedFinalReview = useTypewriter(review?.final_review ?? '', isLoading)

  /* ==================== LANDING STATE ==================== */

  if (!hasResults && !isLoading) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-neutral-50 px-4">
        <div className="w-full max-w-lg text-center">
          <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-500 shadow-lg shadow-indigo-200">
            <Sparkles className="h-7 w-7 text-white" />
          </div>
          <h1 className="mt-6 text-2xl font-semibold tracking-tight text-neutral-900">
            AI PR Review
          </h1>
          <p className="mt-2 text-sm text-neutral-500">
            Paste a GitHub pull request URL and get an instant AI-powered code review.
          </p>

          <form className="mt-8 flex gap-3" onSubmit={analyzePR}>
            <Input
              className="flex-1 text-left"
              value={prUrl}
              onChange={(e) => setPrUrl(e.target.value)}
              placeholder="https://github.com/owner/repo/pull/123"
            />
            <Button type="submit" disabled={isLoading || prUrl.trim() === ''} size="lg">
              {isLoading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Analyzing
                </>
              ) : (
                'Analyze'
              )}
            </Button>
          </form>

          {error ? (
            <div className="mt-4 flex items-start gap-2 rounded-lg bg-red-50 px-4 py-3 text-left text-sm text-red-600">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              {error}
            </div>
          ) : null}

          {serviceStatus && !serviceStatus.ai.enabled ? (
            <div className="mt-6 inline-flex items-center gap-1.5 rounded-full border border-amber-200 bg-amber-50 px-3 py-1.5 text-xs text-amber-600">
              <AlertTriangle className="h-3 w-3" />
              AI not configured &mdash; reviews will use fallback data.
            </div>
          ) : null}

          {serviceStatus && serviceStatus.ai.enabled ? (
            <p className="mt-6 text-xs text-neutral-400">
              AI connected &mdash; model <span className="font-medium text-neutral-500">{serviceStatus.ai.base_url}</span>
            </p>
          ) : null}

          <p className="mt-3 text-xs text-neutral-400">
            The default URL points to a public PR for quick demos.
          </p>
        </div>
      </main>
    )
  }

  /* ==================== LOADING SKELETON ==================== */

  if (isLoading && !review) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-neutral-50 px-4">
        <div className="w-full max-w-lg text-center">
          <Loader2 className="mx-auto h-10 w-10 animate-spin text-indigo-400" />
          <p className="mt-4 text-sm font-medium text-neutral-600">
            {statusMessage || 'Preparing...'}
          </p>
          <p className="mt-1 text-xs text-neutral-400">This may take a moment for large pull requests.</p>
        </div>
      </main>
    )
  }

  /* ==================== RESULTS STATE ==================== */

  return (
    <ErrorBoundary>
      <main className="flex h-screen flex-col overflow-hidden bg-neutral-50">
      {/* ---- Top bar ---- */}
      <header className="flex shrink-0 items-center gap-3 border-b border-neutral-200 bg-white px-4 py-3">
        <button
          className="flex h-8 w-8 items-center justify-center rounded-md text-neutral-400 transition hover:bg-neutral-100 hover:text-neutral-600"
          onClick={() => setSidebarOpen((v) => !v)}
          type="button"
          title={sidebarOpen ? 'Collapse sidebar' : 'Expand sidebar'}
        >
          {sidebarOpen ? <PanelLeftClose className="h-4 w-4" /> : <PanelLeft className="h-4 w-4" />}
        </button>

        <div
          className={cn(
            'flex h-9 items-center gap-2 rounded-lg px-3 text-white',
            serviceStatus && !serviceStatus.ai.enabled ? 'bg-amber-500' : 'bg-indigo-500',
          )}
          title={serviceStatus && !serviceStatus.ai.enabled ? 'AI not configured — showing fallback results' : undefined}
        >
          <Sparkles className="h-4 w-4" />
          <span className="text-sm font-medium">AI PR Review</span>
        </div>

        <div className="min-w-0 flex-1">
          <h2 className="truncate text-sm font-medium text-neutral-900">
            {review?.pr.title ?? 'Loading...'}
          </h2>
        </div>

        <div className="hidden shrink-0 items-center gap-4 text-xs text-neutral-400 sm:flex">
          <span className="flex items-center gap-1">
            <FileCode className="h-3 w-3" />
            {review?.pr.files_changed ?? 0} files
          </span>
          <span className="flex items-center gap-1 text-emerald-500">
            <Plus className="h-3 w-3" />{review?.pr.additions ?? 0}
          </span>
          <span className="flex items-center gap-1 text-red-400">
            <Minus className="h-3 w-3" />{review?.pr.deletions ?? 0}
          </span>
          {review && review.risks.length > 0 ? (
            <span className="flex items-center gap-1 text-amber-500">
              <AlertTriangle className="h-3 w-3" />{review.risks.length} risks
            </span>
          ) : null}
        </div>

        {/* mini PR URL input in results bar */}
        <form className="hidden items-center gap-2 lg:flex" onSubmit={analyzePR}>
          <Input
            className="h-8 w-72 text-xs"
            value={prUrl}
            onChange={(e) => setPrUrl(e.target.value)}
            placeholder="PR URL"
          />
          <Button type="submit" size="sm" disabled={isLoading || prUrl.trim() === ''}>
            {isLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : 'Go'}
          </Button>
        </form>
      </header>

      {error ? (
        <div className="mx-4 mt-3 flex items-start gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          {error}
        </div>
      ) : null}

      {/* ---- Body ---- */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left sidebar */}
        <aside
          className={cn(
            'flex shrink-0 flex-col border-r border-neutral-200 bg-white transition-all duration-300',
            sidebarOpen ? 'w-[260px]' : 'w-0 overflow-hidden border-r-0',
          )}
        >
          <div className="w-[260px] flex flex-1 flex-col">
            {/* PR info */}
            <div className="border-b border-neutral-100 p-4">
              <p className="text-xs font-medium uppercase tracking-wider text-neutral-400">PR Info</p>
              <div className="mt-3 space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-neutral-400">Author</span>
                  <span className="text-neutral-700">{review?.pr.author}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-neutral-400">Files</span>
                  <span className="text-neutral-700">{review?.pr.files_changed}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-neutral-400">Changes</span>
                  <span className="text-neutral-700">
                    <span className="text-emerald-500">+{review?.pr.additions}</span>
                    {' / '}
                    <span className="text-red-400">-{review?.pr.deletions}</span>
                  </span>
                </div>
              </div>
            </div>

            {/* File tree */}
            <div className="flex-1 overflow-hidden">
              <p className="px-4 pt-4 text-xs font-medium uppercase tracking-wider text-neutral-400">
                Files
              </p>
              <ScrollArea className="h-full p-2">
                <div className="flex flex-col gap-0.5">
                  {review?.files.map((file) => (
                    <button
                      className={cn(
                        'flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition-colors',
                        file.filename === selectedFile
                          ? 'bg-indigo-50 text-indigo-700'
                          : 'text-neutral-600 hover:bg-neutral-50',
                      )}
                      key={file.filename}
                      onClick={() => toggleFile(file.filename)}
                      type="button"
                    >
                      <ChevronRight
                        className={cn(
                          'h-3 w-3 shrink-0 transition-transform',
                          file.filename === selectedFile && 'rotate-90',
                        )}
                      />
                      <span className="min-w-0 flex-1 truncate font-mono">{file.filename}</span>
                    </button>
                  ))}
                </div>
              </ScrollArea>
            </div>
          </div>
        </aside>

        {/* Center: AI Review */}
        <section className="flex flex-1 flex-col overflow-hidden">
          {/* Tabs */}
          <nav className="flex shrink-0 items-center gap-1 border-b border-neutral-200 bg-white px-4">
            {([
              ['summary', 'Summary', FileText],
              ['risks', 'Risks', AlertTriangle],
              ['comments', 'Comments', MessageSquare],
              ['final', 'Final Review', Copy],
            ] as const).map(([key, label, Icon]) => (
              <button
                className={cn(
                  'flex items-center gap-1.5 border-b-2 px-3 py-2.5 text-xs font-medium transition-colors',
                  tab === key
                    ? 'border-indigo-500 text-indigo-600'
                    : 'border-transparent text-neutral-400 hover:text-neutral-600',
                )}
                key={key}
                onClick={() => setTab(key)}
                type="button"
              >
                <Icon className="h-3.5 w-3.5" />
                {label}
                {key === 'risks' && review && review.risks.length > 0 ? (
                  <span className="ml-0.5 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold text-amber-600">
                    {review.risks.length}
                  </span>
                ) : null}
              </button>
            ))}
          </nav>

          {/* Tab content */}
          <ErrorBoundary>
          <ScrollArea className="flex-1 p-6">
            {tab === 'summary' ? (
              <div className="mx-auto max-w-2xl">
                {review?.summary ? (
                  <div className="relative rounded-xl border border-neutral-200 bg-white p-6 shadow-sm">
                    <div className="absolute left-0 top-4 bottom-4 w-0.5 rounded-full bg-indigo-400" />
                    <div className="flex items-center gap-2 mb-3">
                      <Sparkles className="h-3.5 w-3.5 text-indigo-500" />
                      <span className="text-[11px] font-medium uppercase tracking-wider text-indigo-400">AI Analysis</span>
                    </div>
                    {isLoading ? (
                      <p className="text-[15px] leading-7 text-neutral-700">{typedSummary}</p>
                    ) : (
                      <MarkdownContent>{typedSummary}</MarkdownContent>
                    )}
                  </div>
                ) : isLoading ? (
                  <div className="flex flex-col items-center gap-4 py-16 text-center">
                    <div className="relative flex h-12 w-12 items-center justify-center">
                      <div className="absolute inset-0 animate-ping rounded-full bg-indigo-100 opacity-75" />
                      <Sparkles className="relative h-5 w-5 text-indigo-400" />
                    </div>
                    <p className="text-sm font-medium text-neutral-500">AI is analyzing this PR...</p>
                    <p className="text-xs text-neutral-400">This may take a moment for large pull requests.</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-3 py-16 text-center">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100">
                      <FileText className="h-5 w-5 text-neutral-300" />
                    </div>
                    <p className="text-sm text-neutral-500">No summary generated.</p>
                  </div>
                )}
              </div>
            ) : tab === 'risks' ? (
              <div className="mx-auto max-w-2xl">
                {review && review.risks.length > 0 ? (
                  <div className="flex flex-col gap-3">
                    <div className="flex items-center gap-2 px-1">
                      <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />
                      <span className="text-xs font-medium text-neutral-500">{review.risks.length} risk{review.risks.length > 1 ? 's' : ''} identified</span>
                    </div>
                    {review.risks.map((risk, i) => {
                      const fileExists = Array.isArray(review.files) && review.files.some((f) => f.filename === risk.file)
                      return (
                        <div
                          className={cn(
                            'relative overflow-hidden rounded-xl border bg-white shadow-sm transition-shadow hover:shadow-md',
                            risk.level === 'high'
                              ? 'border-red-200'
                              : risk.level === 'medium'
                                ? 'border-amber-200'
                                : 'border-emerald-200',
                          )}
                          key={i}
                        >
                          <div
                            className={cn(
                              'absolute inset-y-0 left-0 w-1',
                              risk.level === 'high'
                                ? 'bg-red-400'
                                : risk.level === 'medium'
                                  ? 'bg-amber-400'
                                  : 'bg-emerald-400',
                            )}
                          />
                          <div className="py-4 pl-5 pr-5">
                            <div className="flex items-start justify-between gap-4">
                              <div className="min-w-0">
                                <div className="flex items-center gap-2">
                                  <span
                                    className={cn(
                                      'shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide',
                                      risk.level === 'high'
                                        ? 'bg-red-100 text-red-600'
                                        : risk.level === 'medium'
                                          ? 'bg-amber-100 text-amber-600'
                                          : 'bg-emerald-100 text-emerald-600',
                                    )}
                                  >
                                    {risk.level}
                                  </span>
                                  <h4 className="text-sm font-semibold text-neutral-900">{risk.title}</h4>
                                </div>
                                {risk.file ? (
                                  fileExists ? (
                                    <button
                                      className="mt-2 inline-flex items-center gap-1.5 rounded-md bg-neutral-100 px-2.5 py-1 font-mono text-xs text-neutral-500 transition hover:bg-indigo-50 hover:text-indigo-600"
                                      onClick={() => setSelectedFile(risk.file)}
                                      type="button"
                                    >
                                      <FileCode className="h-3 w-3" />
                                      {risk.file}:{risk.line}
                                    </button>
                                  ) : (
                                    <span className="mt-2 inline-flex items-center gap-1.5 rounded-md bg-neutral-100 px-2.5 py-1 font-mono text-xs text-neutral-400">
                                      <FileCode className="h-3 w-3" />
                                      {risk.file}:{risk.line}
                                    </span>
                                  )
                                ) : null}
                              </div>
                            </div>
                            <p className="mt-3 text-sm leading-6 text-neutral-600">{risk.description}</p>
                            <div className="mt-3 flex items-start gap-2.5 rounded-lg border border-indigo-100 bg-indigo-50/50 px-4 py-2.5">
                              <span className="mt-0.5 shrink-0 text-xs font-bold text-indigo-400">→</span>
                              <span className="text-sm leading-6 text-indigo-700">{risk.suggestion}</span>
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                ) : isLoading ? (
                  <div className="flex flex-col items-center gap-4 py-16 text-center">
                    <div className="relative flex h-12 w-12 items-center justify-center">
                      <div className="absolute inset-0 animate-ping rounded-full bg-amber-100 opacity-75" />
                      <AlertTriangle className="relative h-5 w-5 text-amber-400" />
                    </div>
                    <p className="text-sm font-medium text-neutral-500">Scanning for risks...</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-3 py-16 text-center">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-emerald-50">
                      <Check className="h-5 w-5 text-emerald-400" />
                    </div>
                    <p className="text-sm font-medium text-neutral-500">No risks identified</p>
                    <p className="text-xs text-neutral-400">The AI didn't flag any issues in this PR.</p>
                  </div>
                )}
              </div>
            ) : tab === 'comments' ? (
              <div className="mx-auto max-w-2xl">
                {review && review.review_comments.length > 0 ? (
                  <div className="flex flex-col gap-3">
                    <div className="flex items-center gap-2 px-1">
                      <MessageSquare className="h-3.5 w-3.5 text-indigo-500" />
                      <span className="text-xs font-medium text-neutral-500">{review.review_comments.length} comment{review.review_comments.length > 1 ? 's' : ''}</span>
                    </div>
                    {review.review_comments.map((c, i) => {
                      const fileExists = Array.isArray(review.files) && review.files.some((f) => f.filename === c.file)
                      return (
                        <div
                          className="group relative rounded-xl border border-neutral-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md"
                          key={i}
                        >
                          <div className="flex items-start gap-3">
                            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-indigo-50 text-[11px] font-semibold text-indigo-500">
                              {i + 1}
                            </span>
                            <div className="min-w-0 flex-1">
                              {c.file ? (
                                fileExists ? (
                                  <button
                                    className="inline-flex items-center gap-1.5 rounded-md bg-neutral-100 px-2.5 py-1 font-mono text-xs text-neutral-500 transition hover:bg-indigo-50 hover:text-indigo-600"
                                    onClick={() => setSelectedFile(c.file)}
                                    type="button"
                                  >
                                    <FileCode className="h-3 w-3" />
                                    {c.file}:{c.line}
                                  </button>
                                ) : (
                                  <span className="inline-flex items-center gap-1.5 rounded-md bg-neutral-100 px-2.5 py-1 font-mono text-xs text-neutral-400">
                                    <FileCode className="h-3 w-3" />
                                    {c.file}:{c.line}
                                  </span>
                                )
                              ) : null}
                              <p className="mt-3 text-sm leading-6 text-neutral-700">{c.comment}</p>
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                ) : isLoading ? (
                  <div className="flex flex-col items-center gap-4 py-16 text-center">
                    <div className="relative flex h-12 w-12 items-center justify-center">
                      <div className="absolute inset-0 animate-ping rounded-full bg-indigo-100 opacity-75" />
                      <MessageSquare className="relative h-5 w-5 text-indigo-400" />
                    </div>
                    <p className="text-sm font-medium text-neutral-500">Generating review comments...</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-3 py-16 text-center">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100">
                      <MessageSquare className="h-5 w-5 text-neutral-300" />
                    </div>
                    <p className="text-sm font-medium text-neutral-500">No review comments</p>
                    <p className="text-xs text-neutral-400">Comments will appear here once generated by the AI.</p>
                  </div>
                )}
              </div>
            ) : (
              <div className="mx-auto max-w-2xl">
                {review?.final_review ? (
                  <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-sm">
                    <div className="flex items-center justify-between gap-4 border-b border-neutral-100 bg-neutral-50/50 px-5 py-3">
                      <div className="flex items-center gap-2">
                        <div className="flex h-2 w-2 rounded-full bg-emerald-400" />
                        <span className="text-xs font-medium text-neutral-500">Ready to paste into GitHub</span>
                      </div>
                      <Button variant="outline" size="sm" onClick={copyFinalReview}>
                        {copied ? (
                          <>
                            <Check className="h-3.5 w-3.5" /> Copied
                          </>
                        ) : (
                          <>
                            <Copy className="h-3.5 w-3.5" /> Copy
                          </>
                        )}
                      </Button>
                    </div>
                    <div className="bg-neutral-50/30 p-5">
                      {isLoading ? (
                        <pre className="whitespace-pre-wrap font-mono text-[13px] leading-7 text-neutral-700">{typedFinalReview}</pre>
                      ) : (
                        <MarkdownContent>{typedFinalReview}</MarkdownContent>
                      )}
                    </div>
                  </div>
                ) : isLoading ? (
                  <div className="flex flex-col items-center gap-4 py-16 text-center">
                    <div className="relative flex h-12 w-12 items-center justify-center">
                      <div className="absolute inset-0 animate-ping rounded-full bg-neutral-200 opacity-75" />
                      <Copy className="relative h-5 w-5 text-neutral-400" />
                    </div>
                    <p className="text-sm font-medium text-neutral-500">Composing final review summary...</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center gap-3 py-16 text-center">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100">
                      <FileText className="h-5 w-5 text-neutral-300" />
                    </div>
                    <p className="text-sm font-medium text-neutral-500">No final review yet</p>
                    <p className="text-xs text-neutral-400">The final summary will appear here once ready.</p>
                  </div>
                )}
              </div>
            )}
          </ScrollArea>
          </ErrorBoundary>
        </section>

        {/* Right diff panel overlay + slide-in */}
        <ErrorBoundary>
        {diffOpen ? (
          <div className="relative z-20 flex shrink-0">
            {/* backdrop */}
            <div
              className="fixed inset-0 bg-neutral-950/20 animate-fade-in"
              onClick={() => setSelectedFile('')}
            />
            {/* panel */}
            <aside className="relative flex w-[520px] shrink-0 flex-col border-l border-neutral-200 bg-white shadow-lg animate-slide-in">
              <div className="flex items-center justify-between gap-3 border-b border-neutral-100 px-4 py-3">
                <div className="min-w-0">
                  <p className="text-xs font-medium uppercase tracking-wider text-neutral-400">Diff</p>
                  <h2 className="mt-0.5 truncate font-mono text-sm text-neutral-700">
                    {activeFile?.filename}
                  </h2>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-neutral-400">
                    <span className="text-emerald-500">+{activeFile?.additions}</span>
                    {' '}
                    <span className="text-red-400">-{activeFile?.deletions}</span>
                  </span>
                  <button
                    className="flex h-7 w-7 items-center justify-center rounded-md text-neutral-400 transition hover:bg-neutral-100 hover:text-neutral-600"
                    onClick={() => setSelectedFile('')}
                    type="button"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              </div>
              <ScrollArea className="flex-1">
                {activeFile?.patch ? (
                  <pre className="min-w-full p-4 font-mono text-xs leading-5 text-neutral-600">
                    {patchLines.map((line, index) => (
                      <code
                        className={cn(
                          'block whitespace-pre px-2',
                          line.startsWith('+') && 'bg-emerald-50 text-emerald-700',
                          line.startsWith('-') && 'bg-red-50 text-red-700',
                          line.startsWith('@@') && 'bg-indigo-50 text-indigo-600',
                        )}
                        key={`${index}:${line.slice(0, 20)}`}
                      >
                        {line || ' '}
                      </code>
                    ))}
                  </pre>
                ) : (
                  <p className="p-6 text-sm text-neutral-400">
                    No patch available for this file.
                  </p>
                )}
              </ScrollArea>
            </aside>
          </div>
        ) : null}
          </ErrorBoundary>
      </div>
    </main>
    </ErrorBoundary>
  )
}

export default App
