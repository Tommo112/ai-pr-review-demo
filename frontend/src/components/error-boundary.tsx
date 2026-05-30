import { Component } from 'react'
import type { ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'

type Props = { children: ReactNode }
type State = { error: string | null }

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(err: Error): State {
    return { error: err.message }
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex min-h-[400px] items-center justify-center px-6">
          <div className="text-center">
            <AlertTriangle className="mx-auto h-8 w-8 text-red-400" />
            <p className="mt-3 text-sm font-medium text-neutral-600">Something went wrong rendering this section.</p>
            <p className="mt-1 text-xs text-neutral-400">{this.state.error}</p>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
