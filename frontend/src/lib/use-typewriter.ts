import { useState, useEffect, useRef } from 'react'

export function useTypewriter(text: string, enabled: boolean): string {
  const [displayed, setDisplayed] = useState('')
  const textRef = useRef(text)
  textRef.current = text

  // Reset when analysis starts fresh or streaming ends
  useEffect(() => {
    if (!enabled) {
      setDisplayed(text)
    } else if (text === '') {
      setDisplayed('')
    }
  }, [text, enabled])

  useEffect(() => {
    if (!enabled) return
    if (displayed.length >= textRef.current.length) return

    let lastTick = performance.now()
    let raf = 0

    function tick(now: number) {
      const elapsed = now - lastTick
      lastTick = now
      const charsPerSec = 120
      const charsToAdd = Math.max(1, Math.floor((elapsed / 1000) * charsPerSec))

      setDisplayed((prev) => {
        const target = textRef.current
        const nextLen = Math.min(prev.length + charsToAdd, target.length)
        return target.slice(0, nextLen)
      })

      raf = requestAnimationFrame(tick)
    }

    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [enabled, text.length, displayed.length])

  if (!enabled) return text
  return displayed
}
