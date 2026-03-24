import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export default function Login() {
  const { login, user } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [needsSetup, setNeedsSetup] = useState(false)
  const [isCheckingSetup, setIsCheckingSetup] = useState(true)

  const [setupEmail, setSetupEmail] = useState('')
  const [setupPassword, setSetupPassword] = useState('')
  const [setupConfirmPassword, setSetupConfirmPassword] = useState('')
  const [setupError, setSetupError] = useState('')
  const [isCreatingAccount, setIsCreatingAccount] = useState(false)

  useEffect(() => {
    const checkSetup = async () => {
      try {
        const res = await fetch('/api/setup/status')
        const data = await res.json() as { needsSetup: boolean }
        setNeedsSetup(data.needsSetup)
      } catch {
        setNeedsSetup(false)
      } finally {
        setIsCheckingSetup(false)
      }
    }
    void checkSetup()
  }, [])

  if (user) {
    void navigate('/', { replace: true })
    return null
  }

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await login(email, password)
      void navigate('/', { replace: true })
    } catch {
      setError('Invalid email or password. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleSetup = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setSetupError('')
    if (setupPassword !== setupConfirmPassword) {
      setSetupError('Passwords do not match.')
      return
    }
    setIsCreatingAccount(true)
    try {
      const res = await fetch('/api/setup/init', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: setupEmail, password: setupPassword }),
      })
      if (!res.ok) {
        const data = await res.json() as { error?: string }
        setSetupError(data.error ?? 'Failed to create account.')
        return
      }
      await login(setupEmail, setupPassword)
      void navigate('/', { replace: true })
    } catch {
      setSetupError('Failed to create account. Please try again.')
    } finally {
      setIsCreatingAccount(false)
    }
  }

  if (isCheckingSetup) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-muted border-t-primary" />
      </div>
    )
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <h1 className="text-3xl font-bold tracking-tight">
            Red<span className="text-red-500">Veluvanto</span>
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {needsSetup ? "Welcome! Let's set up your instance." : 'Reddit Copilot — sign in to continue'}
          </p>
        </div>

        {needsSetup ? (
          <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
            <div className="mb-4 rounded-md bg-primary/10 px-4 py-3">
              <h2 className="text-sm font-semibold mb-1">First-Time Setup</h2>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Create your admin account to get started.
              </p>
            </div>

            <form onSubmit={(e) => void handleSetup(e)} className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="setup-email">Email</Label>
                <Input
                  id="setup-email"
                  type="email"
                  autoComplete="email"
                  required
                  value={setupEmail}
                  onChange={(e) => setSetupEmail(e.target.value)}
                  placeholder="you@example.com"
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="setup-password">Password</Label>
                <Input
                  id="setup-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  value={setupPassword}
                  onChange={(e) => setSetupPassword(e.target.value)}
                  placeholder="••••••••"
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="setup-confirm-password">Confirm Password</Label>
                <Input
                  id="setup-confirm-password"
                  type="password"
                  autoComplete="new-password"
                  required
                  value={setupConfirmPassword}
                  onChange={(e) => setSetupConfirmPassword(e.target.value)}
                  placeholder="••••••••"
                />
              </div>

              {setupError && (
                <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {setupError}
                </p>
              )}

              <Button type="submit" className="w-full mt-1" disabled={isCreatingAccount}>
                {isCreatingAccount ? 'Creating account…' : 'Create Account'}
              </Button>
            </form>

            <p className="mt-4 text-center text-xs text-muted-foreground">
              Already have an account?{' '}
              <button
                type="button"
                className="underline hover:text-foreground transition-colors"
                onClick={() => setNeedsSetup(false)}
              >
                Sign in
              </button>
            </p>
          </div>
        ) : (
          <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
            <form onSubmit={(e) => void handleSubmit(e)} className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                />
              </div>

              {error && (
                <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {error}
                </p>
              )}

              <Button type="submit" className="w-full mt-1" disabled={isSubmitting}>
                {isSubmitting ? 'Signing in…' : 'Sign in'}
              </Button>
            </form>
          </div>
        )}

        <p className="mt-6 text-center text-xs text-muted-foreground">
          Made with ❤️ by the{' '}
          <a
            href="https://veluvanto.com"
            target="_blank"
            rel="noopener noreferrer"
            className="underline hover:text-foreground transition-colors"
          >
            Veluvanto team
          </a>
        </p>
      </div>
    </div>
  )
}
