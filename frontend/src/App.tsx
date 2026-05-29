import { useState } from 'react'
import type { FormEvent } from 'react'

type PullRequestFile = {
  filename: string
  status: string
  additions: number
  deletions: number
  patch: string
}

type ReviewResponse = {
  pr: {
    title: string
    author: string
    files_changed: number
    additions: number
    deletions: number
  }
  files: PullRequestFile[]
  summary: string
  risks: Array<{
    level: string
    file: string
    line: number
    title: string
    description: string
    suggestion: string
  }>
  review_comments: Array<{
    file: string
    line: number
    comment: string
  }>
  final_review: string
}

function App() {
  const [prUrl, setPrUrl] = useState('https://github.com/owner/repo/pull/1')
  const [review, setReview] = useState<ReviewResponse | null>(null)
  const [selectedFile, setSelectedFile] = useState('')
  const [error, setError] = useState('')
  const [copyStatus, setCopyStatus] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  async function analyzePR(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsLoading(true)
    setError('')

    try {
      const response = await fetch('/api/review', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pr_url: prUrl }),
      })

      if (!response.ok) {
        const errorBody = (await response.json().catch(() => null)) as { error?: string } | null
        throw new Error(errorBody?.error ?? `Request failed with ${response.status}`)
      }

      const data = (await response.json()) as ReviewResponse
      setReview(data)
      setSelectedFile(data.files[0]?.filename ?? '')
      setCopyStatus('')
    } catch (err) {
      setReview(null)
      setSelectedFile('')
      setCopyStatus('')
      setError(err instanceof Error ? err.message : 'Request failed')
    } finally {
      setIsLoading(false)
    }
  }

  async function copyFinalReview() {
    if (!review) {
      return
    }

    try {
      await navigator.clipboard.writeText(review.final_review)
      setCopyStatus('Copied')
    } catch {
      setCopyStatus('Copy failed')
    }
  }

  const activeFile = review?.files.find((file) => file.filename === selectedFile)
  const patchLines = activeFile?.patch?.split('\n') ?? []

  return (
    <main className="min-h-screen bg-neutral-950 px-6 py-8 text-neutral-100">
      <section className="mx-auto flex max-w-7xl flex-col gap-6">
        <header className="flex flex-col gap-4 border-b border-neutral-800 pb-6">
          <div>
            <p className="text-sm text-emerald-300">AI PR Review Assistant</p>
            <h1 className="mt-2 text-3xl font-semibold tracking-normal text-white">
              PR review demo workspace
            </h1>
          </div>

          <form className="flex flex-col gap-3 md:flex-row" onSubmit={analyzePR}>
            <input
              className="min-h-11 flex-1 rounded-md border border-neutral-700 bg-neutral-900 px-4 text-sm text-white outline-none transition focus:border-emerald-300"
              value={prUrl}
              onChange={(event) => setPrUrl(event.target.value)}
              placeholder="https://github.com/owner/repo/pull/123"
            />
            <button
              className="min-h-11 rounded-md bg-emerald-300 px-5 text-sm font-medium text-neutral-950 transition hover:bg-emerald-200 disabled:cursor-not-allowed disabled:opacity-60"
              type="submit"
              disabled={isLoading || prUrl.trim() === ''}
            >
              {isLoading ? 'Analyzing...' : 'Analyze'}
            </button>
          </form>

          {isLoading ? (
            <p className="text-sm text-neutral-400">
              Fetching GitHub files and preparing review output...
            </p>
          ) : null}
          {error ? <p className="text-sm text-red-300">{error}</p> : null}
        </header>

        {isLoading && !review ? (
          <section className="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)_360px]">
            {[0, 1, 2].map((item) => (
              <div
                className="min-h-[420px] rounded-md border border-neutral-800 bg-neutral-900 p-4"
                key={item}
              >
                <div className="h-3 w-24 animate-pulse rounded bg-neutral-800" />
                <div className="mt-6 h-5 w-2/3 animate-pulse rounded bg-neutral-800" />
                <div className="mt-4 space-y-3">
                  <div className="h-3 animate-pulse rounded bg-neutral-800" />
                  <div className="h-3 w-5/6 animate-pulse rounded bg-neutral-800" />
                  <div className="h-3 w-3/4 animate-pulse rounded bg-neutral-800" />
                </div>
              </div>
            ))}
          </section>
        ) : review ? (
          <section className="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)_360px]">
            <aside className="flex min-h-[620px] flex-col rounded-md border border-neutral-800 bg-neutral-900">
              <div className="border-b border-neutral-800 p-4">
                <p className="text-xs uppercase tracking-widest text-neutral-500">PR</p>
                <h2 className="mt-3 text-lg font-semibold tracking-normal text-white">
                  {review.pr.title}
                </h2>
                <dl className="mt-4 grid gap-3 text-sm">
                  <div className="flex justify-between gap-3">
                    <dt className="text-neutral-400">Author</dt>
                    <dd>{review.pr.author}</dd>
                  </div>
                  <div className="flex justify-between gap-3">
                    <dt className="text-neutral-400">Files</dt>
                    <dd>{review.pr.files_changed}</dd>
                  </div>
                  <div className="flex justify-between gap-3">
                    <dt className="text-neutral-400">Changes</dt>
                    <dd>
                      +{review.pr.additions} / -{review.pr.deletions}
                    </dd>
                  </div>
                </dl>
              </div>

              <div className="min-h-0 flex-1 overflow-auto p-2">
                <p className="px-2 pb-2 pt-1 text-xs uppercase tracking-widest text-neutral-500">
                  Files
                </p>
                <div className="flex flex-col gap-1">
                  {review.files.length > 0 ? (
                    review.files.map((file) => (
                    <button
                      className={`rounded-md px-3 py-2 text-left text-sm transition ${
                        file.filename === selectedFile
                          ? 'bg-emerald-300 text-neutral-950'
                          : 'text-neutral-300 hover:bg-neutral-800'
                      }`}
                      key={file.filename}
                      onClick={() => setSelectedFile(file.filename)}
                      type="button"
                    >
                      <span className="block truncate">{file.filename}</span>
                      <span className="mt-1 block text-xs opacity-75">
                        {file.status} +{file.additions} -{file.deletions}
                      </span>
                    </button>
                    ))
                  ) : (
                    <p className="rounded-md border border-neutral-800 bg-neutral-950 p-3 text-sm text-neutral-500">
                      No changed files returned by GitHub.
                    </p>
                  )}
                </div>
              </div>
            </aside>

            <section className="min-h-[620px] rounded-md border border-neutral-800 bg-neutral-900">
              <div className="flex items-center justify-between gap-3 border-b border-neutral-800 px-4 py-3">
                <div className="min-w-0">
                  <p className="text-xs uppercase tracking-widest text-neutral-500">Diff</p>
                  <h2 className="mt-1 truncate text-sm font-medium tracking-normal text-white">
                    {activeFile?.filename ?? 'No file selected'}
                  </h2>
                </div>
                {activeFile ? (
                  <span className="shrink-0 rounded bg-neutral-800 px-2 py-1 text-xs text-neutral-300">
                    +{activeFile.additions} -{activeFile.deletions}
                  </span>
                ) : null}
              </div>

              <div className="max-h-[560px] overflow-auto">
                {activeFile?.patch ? (
                  <pre className="min-w-full p-4 text-left font-mono text-xs leading-5 text-neutral-300">
                    {patchLines.map((line, index) => (
                      <code
                        className={`block whitespace-pre px-2 ${
                          line.startsWith('+')
                            ? 'bg-emerald-400/10 text-emerald-200'
                            : line.startsWith('-')
                              ? 'bg-red-400/10 text-red-200'
                              : line.startsWith('@@')
                                ? 'bg-sky-400/10 text-sky-200'
                                : ''
                        }`}
                        key={`${index}:${line}`}
                      >
                        {line || ' '}
                      </code>
                    ))}
                  </pre>
                ) : (
                  <p className="p-4 text-sm text-neutral-500">
                    This file has no patch available from GitHub.
                  </p>
                )}
              </div>
            </section>

            <aside className="flex min-h-[620px] flex-col gap-4 rounded-md border border-neutral-800 bg-neutral-900 p-4">
              <section>
                <p className="text-xs uppercase tracking-widest text-neutral-500">Summary</p>
                <p className="mt-3 text-sm leading-6 text-neutral-200">{review.summary}</p>
              </section>

              <section>
                <p className="text-xs uppercase tracking-widest text-neutral-500">Risks</p>
                <div className="mt-4 flex flex-col gap-3">
                  {review.risks.length > 0 ? (
                    review.risks.map((risk) => (
                    <article
                      className="rounded-md border border-neutral-800 bg-neutral-950 p-3"
                      key={`${risk.file}:${risk.line}:${risk.title}`}
                    >
                      <div className="flex items-center justify-between gap-3">
                        <h3 className="text-sm font-medium text-white">{risk.title}</h3>
                        <span className="rounded bg-amber-300 px-2 py-1 text-xs font-medium uppercase text-neutral-950">
                          {risk.level}
                        </span>
                      </div>
                      <p className="mt-2 text-xs text-neutral-500">
                        {risk.file}:{risk.line}
                      </p>
                      <p className="mt-2 text-sm leading-5 text-neutral-300">
                        {risk.description}
                      </p>
                      <p className="mt-2 text-sm leading-5 text-neutral-400">
                        {risk.suggestion}
                      </p>
                    </article>
                    ))
                  ) : (
                    <p className="rounded-md border border-neutral-800 bg-neutral-950 p-3 text-sm text-neutral-500">
                      No structured risks returned.
                    </p>
                  )}
                </div>
              </section>

              <section>
                <p className="text-xs uppercase tracking-widest text-neutral-500">
                  Review Comments
                </p>
                <div className="mt-4 flex flex-col gap-3">
                  {review.review_comments.length > 0 ? (
                    review.review_comments.map((comment) => (
                      <article
                        className="rounded-md border border-neutral-800 bg-neutral-950 p-3"
                        key={`${comment.file}:${comment.line}:${comment.comment}`}
                      >
                        <p className="text-xs text-neutral-500">
                          {comment.file}:{comment.line}
                        </p>
                        <p className="mt-2 text-sm leading-5 text-neutral-300">
                          {comment.comment}
                        </p>
                      </article>
                    ))
                  ) : (
                    <p className="rounded-md border border-neutral-800 bg-neutral-950 p-3 text-sm text-neutral-500">
                      No review comments returned.
                    </p>
                  )}
                </div>
              </section>

              <section className="min-h-0 flex-1">
                <div className="flex items-center justify-between gap-3">
                  <p className="text-xs uppercase tracking-widest text-neutral-500">
                    Review Summary
                  </p>
                  <button
                    className="rounded bg-neutral-800 px-3 py-1 text-xs text-neutral-200 transition hover:bg-neutral-700"
                    onClick={copyFinalReview}
                    type="button"
                  >
                    Copy
                  </button>
                </div>
                <pre className="mt-3 max-h-52 overflow-auto rounded-md bg-neutral-950 p-4 text-left text-xs leading-6 text-neutral-300">
                  {review.final_review}
                </pre>
                {copyStatus ? (
                  <p className="mt-2 text-xs text-neutral-500">{copyStatus}</p>
                ) : null}
              </section>
            </aside>
          </section>
        ) : (
          <section className="rounded-md border border-neutral-800 bg-neutral-900 p-8 text-left">
            <p className="text-sm text-neutral-400">
              Submit a PR URL to verify the frontend can call the Go API.
            </p>
          </section>
        )}
      </section>
    </main>
  )
}

export default App
