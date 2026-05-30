import { useState, useEffect, useRef } from 'react'

export function useTypewriter(text: string, enabled: boolean): string {
  const [displayed, setDisplayed] = useState('')
  const textRef = useRef(text)
  textRef.current = text

  const finishingRef = useRef(false)
  const enabledRef = useRef(enabled)
  enabledRef.current = enabled

  useEffect(() => {
    if (text === '') {
      setDisplayed('')
      finishingRef.current = false
    }
  }, [text])

  useEffect(() => {
    if (displayed.length >= textRef.current.length && finishingRef.current) {
      finishingRef.current = false
    }
    if (!enabledRef.current && !finishingRef.current && displayed.length >= textRef.current.length) {
      return
    }

    let lastTick = performance.now()
    let raf = 0

    function tick(now: number) {
      const elapsed = now - lastTick
      lastTick = now
      const charsPerSec = finishingRef.current ? 800 : 120
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
  }, [displayed.length, text.length])

  // When streaming ends, enter finishing phase instead of jumping
  useEffect(() => {
    if (!enabled) {
      finishingRef.current = true
    }
  }, [enabled])

  if (!enabled && displayed.length >= text.length) return text
  return displayed
}
