import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import type { RecordModel } from 'pocketbase'
import pb from '@/lib/pocketbase'
import { useAuth } from '@/lib/auth'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { MessageSquare, CheckCircle2, Key, Users, ArrowRight, Clock, Send, XCircle, Loader2 } from 'lucide-react'

interface DraftWithExpand extends RecordModel {
  status: string
  expand?: {
    thread?: {
      id: string
      title: string
    }
  }
}

interface QueueItem {
  id: string
  thread_id: string
  thread_title: string
  subreddit: string
  status: string
  queued_at: string
  text_preview: string
}

interface Stats {
  newThreads: number
  replied: number
  activeKeywords: number
  personas: number
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.floor(hrs / 24)
  return `${days}d ago`
}

export default function Dashboard() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [stats, setStats] = useState<Stats>({ newThreads: 0, replied: 0, activeKeywords: 0, personas: 0 })
  const [recentDrafts, setRecentDrafts] = useState<DraftWithExpand[]>([])
  const [queueItems, setQueueItems] = useState<QueueItem[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      if (!user?.id) return
      setIsLoading(true)
      try {
        const oneDayAgo = new Date(Date.now() - 86400000).toISOString()
        const [newThreadsResult, postedDraftsResult, keywordsResult, personasResult, recentDraftsResult] =
          await Promise.all([
            pb.collection('threads').getFullList({
              filter: `found_at >= "${oneDayAgo}"`,
              fields: 'id',
            }),
            pb.collection('drafts').getFullList({
              filter: `status = "posted" && posted_at >= "${oneDayAgo}" && user = "${user.id}"`,
              fields: 'id',
            }),
            pb.collection('keywords').getFullList({
              filter: 'is_active = true',
              fields: 'id',
            }),
            pb.collection('personas').getFullList({
              filter: `created_by = "${user.id}"`,
              fields: 'id',
            }),
            pb.collection('drafts').getList<DraftWithExpand>(1, 10, {
              filter: `user = "${user.id}"`,
              sort: '-created',
              expand: 'thread',
            }),
          ])
        setStats({
          newThreads: newThreadsResult.length,
          replied: postedDraftsResult.length,
          activeKeywords: keywordsResult.length,
          personas: personasResult.length,
        })
        setRecentDrafts(recentDraftsResult.items)

        try {
          const queueRes = await fetch('/api/drafts/queue', {
            headers: { Authorization: pb.authStore.token },
          })
          if (queueRes.ok) {
            setQueueItems(await queueRes.json() as QueueItem[])
          }
        } catch (_e) {
          void _e
        }
      } catch (_e) {
        void _e
      } finally {
        setIsLoading(false)
      }
    }
    void fetchData()
  }, [user?.id])

  const handleCancelQueued = async (draftId: string) => {
    try {
      const res = await fetch(`/api/drafts/${draftId}/cancel`, {
        method: 'POST',
        headers: { Authorization: pb.authStore.token },
      })
      if (res.ok) {
        setQueueItems((prev) => prev.filter((q) => q.id !== draftId))
      }
    } catch (_e) {
      void _e
    }
  }

  const statCards = [
    { label: 'New Threads', value: stats.newThreads, icon: MessageSquare, color: 'text-blue-400' },
    { label: 'Replied', value: stats.replied, icon: CheckCircle2, color: 'text-green-400' },
    { label: 'Active Keywords', value: stats.activeKeywords, icon: Key, color: 'text-yellow-400' },
    { label: 'Personas', value: stats.personas, icon: Users, color: 'text-violet-400' },
  ]

  return (
    <div className="p-4 md:p-6 flex flex-col gap-6 max-w-3xl mx-auto">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground mt-1">Last 24 hours overview</p>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {statCards.map(({ label, value, icon: Icon, color }) => (
          <Card key={label}>
            <CardContent className="p-4 flex flex-col gap-2">
              <Icon className={`size-5 ${color}`} />
              <span className="text-3xl font-bold">
                {isLoading ? (
                  <span className="inline-block w-10 h-8 rounded bg-muted animate-pulse" />
                ) : (
                  value
                )}
              </span>
              <span className="text-xs text-muted-foreground">{label}</span>
            </CardContent>
          </Card>
        ))}
      </div>

      {queueItems.length > 0 && (
        <div>
          <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3 flex items-center gap-2">
            <Send className="size-3.5" />
            Post Queue
            <Badge variant="outline" className="text-xs">{queueItems.length}</Badge>
          </h2>
          <div className="flex flex-col gap-2">
            {queueItems.map((item) => (
              <div
                key={item.id}
                className="flex items-center gap-3 rounded-lg border border-border bg-card px-3 py-2.5 hover:bg-muted/30 transition-colors"
              >
                {item.status === 'posting' ? (
                  <Loader2 className="size-3.5 text-yellow-400 animate-spin shrink-0" />
                ) : (
                  <Clock className="size-3.5 text-muted-foreground shrink-0" />
                )}
                <div
                  className="flex-1 min-w-0 cursor-pointer"
                  onClick={() => void navigate(`/threads/${item.thread_id}`)}
                >
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="text-xs shrink-0">r/{item.subreddit}</Badge>
                    <span className="text-xs font-medium truncate">{item.thread_title}</span>
                  </div>
                  <p className="text-xs text-muted-foreground truncate mt-0.5">{item.text_preview}</p>
                </div>
                <span className="text-xs text-muted-foreground shrink-0">{timeAgo(item.queued_at)}</span>
                <Badge variant={item.status === 'posting' ? 'default' : 'outline'} className="text-xs capitalize shrink-0">
                  {item.status === 'posting' ? 'Posting…' : 'Queued'}
                </Badge>
                {item.status === 'queued' && (
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => void handleCancelQueued(item.id)}
                    className="shrink-0 text-muted-foreground hover:text-destructive"
                  >
                    <XCircle className="size-3.5" />
                  </Button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      <div>
        <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3">
          Recent Activity
        </h2>
        {isLoading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-12 rounded-lg bg-muted/50 animate-pulse" />
            ))}
          </div>
        ) : recentDrafts.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center rounded-xl border border-border">
            <Clock className="size-8 text-muted-foreground/40 mb-2" />
            <p className="text-sm text-muted-foreground">No recent activity</p>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {recentDrafts.map((draft) => {
              const thread = draft.expand?.thread
              const action = draft.status === 'posted' ? 'Posted to' : 'Generated reply for'
              return (
                <div
                  key={draft.id}
                  className="flex items-center gap-3 rounded-lg border border-border bg-card px-3 py-2.5 hover:bg-muted/30 transition-colors cursor-pointer"
                  onClick={() => thread && void navigate(`/threads/${thread.id}`)}
                >
                  <span className="text-xs text-muted-foreground shrink-0 w-14">
                    {timeAgo(draft.created)}
                  </span>
                  <span className="text-xs text-muted-foreground shrink-0">{action}</span>
                  <span className="text-xs font-medium truncate flex-1">
                    {thread?.title ?? 'Unknown thread'}
                  </span>
                  <ArrowRight className="size-3 text-muted-foreground shrink-0" />
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div>
        <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3">
          Quick Access
        </h2>
        <div className="flex gap-3">
          <Button variant="outline" onClick={() => void navigate('/inbox')} className="flex-1 gap-2">
            <MessageSquare className="size-4" />
            View Inbox
          </Button>
          <Button variant="outline" onClick={() => void navigate('/keywords')} className="flex-1 gap-2">
            <Key className="size-4" />
            Add Keyword
          </Button>
        </div>
      </div>
    </div>
  )
}
