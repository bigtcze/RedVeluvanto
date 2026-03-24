import { ExternalLink } from 'lucide-react'
import { Badge } from '@/components/ui/badge'

export default function About() {
  return (
    <div className="flex flex-col h-full">
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-4">
        <h1 className="text-xl font-bold">About</h1>
      </div>

      <div className="flex-1 overflow-auto p-4 flex flex-col gap-4 max-w-lg">
        <div className="rounded-xl border border-border bg-card p-6 flex flex-col gap-4">
          <div className="flex items-center gap-3">
            <span className="text-2xl font-bold tracking-tight">
              Red<span className="text-red-500">Veluvanto</span>
            </span>
            <Badge variant="secondary" className="text-xs font-mono">v0.1.0</Badge>
          </div>

          <p className="text-sm text-muted-foreground leading-relaxed">
            Open-source Reddit copilot with a persona engine. Monitor keywords, get AI-scored threads with full context, craft replies in your custom persona, and post — all with a human in the loop.
          </p>

          <div className="rounded-lg border border-border bg-muted/20 px-4 py-3">
            <p className="text-xs text-muted-foreground leading-relaxed">
              Built by the team behind{' '}
              <a
                href="https://veluvanto.com"
                target="_blank"
                rel="noopener noreferrer"
                className="text-foreground font-medium hover:text-primary transition-colors"
              >
                Veluvanto
              </a>
              {' '}— AI-native document management.
            </p>
          </div>

          <div className="flex flex-col gap-2">
            <a
              href="https://veluvanto.com"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <ExternalLink className="size-4 shrink-0" />
              veluvanto.com
            </a>
            <a
              href="https://github.com/bigtcze/RedVeluvanto"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <ExternalLink className="size-4 shrink-0" />
              github.com/bigtcze/RedVeluvanto
            </a>
          </div>

          <div className="border-t border-border pt-3 flex items-center justify-between">
            <span className="text-xs text-muted-foreground">License</span>
            <Badge variant="outline" className="text-xs font-mono">MIT</Badge>
          </div>
        </div>
      </div>
    </div>
  )
}
