import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import type { RecordModel } from 'pocketbase'
import pb from '@/lib/pocketbase'
import { useAuth } from '@/lib/auth'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { MessageSquare, CheckCircle2, Key, Users, ArrowRight, Clock } from 'lucide-react'

interface DraftWithExpand extends RecordModel {
  status: string
  expand?: {
    thread?: {
      id: string
      title: string
    }
  }
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
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      if (!user?.id) return
      setIsLoading(true)
      try {
        const oneDayAgo = new Date(Date.now() - 86400000).toISOString()
        const [newThreadsResult, postedDraftsResult, keywordsResult, personasResult, recentDraftsResult] =
          await Promise.all([
            pb.collection('threads').getList(1, 1, {
              filter: `found_at >= "${oneDayAgo}"`,
            }),
            pb.collection('drafts').getList(1, 1, {
              filter: `status = "posted" && posted_at >= "${oneDayAgo}" && user = "${user.id}"`,
            }),
            pb.collection('keywords').getList(1, 1, {
              filter: 'is_active = true',
            }),
            pb.collection('personas').getList(1, 1, {
              filter: `created_by = "${user.id}"`,
            }),
            pb.collection('drafts').getList<DraftWithExpand>(1, 10, {
              filter: `user = "${user.id}"`,
              sort: '-created',
              expand: 'thread',
            }),
          ])
        setStats({
          newThreads: newThreadsResult.totalItems,
          replied: postedDraftsResult.totalItems,
          activeKeywords: keywordsResult.totalItems,
          personas: personasResult.totalItems,
        })
        setRecentDrafts(recentDraftsResult.items)
      } catch {
      } finally {
        setIsLoading(false)
      }
    }
    void fetchData()
  }, [user?.id])

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
