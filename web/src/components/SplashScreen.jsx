import { useState, useEffect } from 'react'
import './SplashScreen.css'

const DECODE_CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$%&'
const TARGET = 'MISSION CONTROL'
const SEQUENCE = [
  { delay: 200,  text: 'INITIALIZING SECURE CHANNEL…' },
  { delay: 550,  text: 'AUTHENTICATING OPERATIVE…' },
  { delay: 950,  text: 'RETRIEVING INTELLIGENCE…' },
  { delay: 1350, text: 'ACCESS GRANTED' },
]

function useDecodeText(target, startDelay = 150) {
  const [display, setDisplay] = useState(() => '█'.repeat(target.length))
  const [done, setDone] = useState(false)

  useEffect(() => {
    let frame
    const startAt = Date.now() + startDelay
    const duration = 700

    function tick() {
      const now = Date.now()
      if (now < startAt) { frame = requestAnimationFrame(tick); return }

      const progress = Math.min(1, (now - startAt) / duration)
      const resolvedChars = Math.floor(progress * target.length)

      const chars = target.split('').map((ch, i) => {
        if (ch === ' ') return ' '
        if (i < resolvedChars) return ch
        return DECODE_CHARS[Math.floor(Math.random() * DECODE_CHARS.length)]
      })

      setDisplay(chars.join(''))

      if (progress < 1) {
        frame = requestAnimationFrame(tick)
      } else {
        setDisplay(target)
        setDone(true)
      }
    }

    frame = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(frame)
  }, [target, startDelay])

  return { display, done }
}

export function SplashScreen({ onDone }) {
  const [statusLines, setStatusLines] = useState([])
  const [fadeOut, setFadeOut] = useState(false)
  const [countdown, setCountdown] = useState(3)
  const { display: titleDisplay, done: titleDone } = useDecodeText(TARGET)

  // Status line typewriter
  useEffect(() => {
    const timers = SEQUENCE.map(({ delay, text }) =>
      setTimeout(() => setStatusLines((l) => [...l, text]), delay)
    )
    return () => timers.forEach(clearTimeout)
  }, [])

  // Countdown tick
  useEffect(() => {
    const interval = setInterval(() => {
      setCountdown((c) => {
        if (c <= 1) { clearInterval(interval); return 0 }
        return c - 1
      })
    }, 650)
    return () => clearInterval(interval)
  }, [])

  // Trigger fade-out + call onDone
  useEffect(() => {
    const timer = setTimeout(() => {
      setFadeOut(true)
      setTimeout(onDone, 400)
    }, 1650)
    return () => clearTimeout(timer)
  }, [onDone])

  return (
    <div className={`splash ${fadeOut ? 'splash-out' : ''}`} aria-live="polite">
      <div className="splash-scanline" aria-hidden />
      <div className="splash-vignette" aria-hidden />

      <div className="splash-content">
        {/* Radar animation */}
        <div className="splash-radar" aria-hidden>
          <div className="radar-ring r1" />
          <div className="radar-ring r2" />
          <div className="radar-ring r3" />
          <div className="radar-sweep" />
          <div className="radar-center" />
        </div>

        {/* Title */}
        <div className="splash-title-wrap">
          <div className="splash-eyebrow mono">CLASSIFIED // EYES ONLY</div>
          <h1 className="splash-title mono">{titleDisplay}</h1>
          <div className={`splash-underline ${titleDone ? 'done' : ''}`} />
        </div>

        {/* Status log */}
        <div className="splash-log" aria-label="Status">
          {statusLines.map((line, i) => (
            <div
              key={i}
              className={`splash-log-line mono ${line === 'ACCESS GRANTED' ? 'granted' : ''}`}
            >
              <span className="log-prompt">{'>'}</span> {line}
              {i === statusLines.length - 1 && line !== 'ACCESS GRANTED' && (
                <span className="log-cursor" aria-hidden>█</span>
              )}
            </div>
          ))}
        </div>

        {/* Countdown */}
        <div className="splash-countdown" aria-hidden>
          <div className="countdown-ring">
            <svg viewBox="0 0 60 60">
              <circle cx="30" cy="30" r="26" className="countdown-track" />
              <circle
                cx="30" cy="30" r="26"
                className="countdown-progress"
                strokeDasharray={`${163.4 * (countdown / 3)} 163.4`}
              />
            </svg>
            <span className="countdown-num mono">{countdown}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
