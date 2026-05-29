import { useState } from 'react'
import type { FormEvent } from 'react'

type ReviewResponse = {
  pr: {
    title: string
    author: string
    files_changed: number
    additions: number
    deletions: number
  }
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
  const [error, setError] = useState('')
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
        throw new Error(`Request failed with ${response.status}`)
      }

      setReview((await response.json()) as ReviewResponse)
    } catch (err) {
      setReview(null)
      setError(err instanceof Error ? err.message : 'Request failed')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <main className="min-h-screen bg-neutral-950 px-6 py-8 text-neutral-100">
      <section className="mx-auto flex max-w-6xl flex-col gap-6">
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
              disabled={isLoading}
            >
              {isLoading ? 'Analyzing...' : 'Analyze'}
            </button>
          </form>

          {error ? <p className="text-sm text-red-300">{error}</p> : null}
        </header>

        {review ? (
          <section className="grid gap-4 lg:grid-cols-[260px_1fr_340px]">
            <aside className="rounded-md border border-neutral-800 bg-neutral-900 p-4">
              <p className="text-xs uppercase tracking-widest text-neutral-500">PR</p>
              <h2 className="mt-3 text-xl font-semibold tracking-normal text-white">
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
            </aside>

            <section className="rounded-md border border-neutral-800 bg-neutral-900 p-4">
              <p className="text-xs uppercase tracking-widest text-neutral-500">Summary</p>
              <p className="mt-3 text-sm leading-6 text-neutral-200">{review.summary}</p>
              <pre className="mt-5 overflow-auto rounded-md bg-neutral-950 p-4 text-left text-xs leading-6 text-neutral-300">
                {review.final_review}
              </pre>
            </section>

            <aside className="rounded-md border border-neutral-800 bg-neutral-900 p-4">
              <p className="text-xs uppercase tracking-widest text-neutral-500">Risks</p>
              <div className="mt-4 flex flex-col gap-3">
                {review.risks.map((risk) => (
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
                  </article>
                ))}
              </div>
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
