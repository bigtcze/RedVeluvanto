import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router'
import type { RecordModel } from 'pocketbase'
import pb from '@/lib/pocketbase'
import { useAuth } from '@/lib/auth'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { MessageSquare, ArrowUp, Clock, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/utils'

type ThreadStatus = 'new' | 'reviewed' | 'replied' | 'dismissed'

interface Thread extends RecordModel {
  title: string
  subreddit: string
  body: string
  author: string
  reddit_score: number
  comment_count: number
  relevance_score: number
  found_at: string
  reddit_created_at?: string
  matched_keyword: string
}

interface StatusRecord extends RecordModel {
  thread: string
  status: ThreadStatus
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

function isStale(thread: Thread): boolean {
  const dateStr = thread.reddit_created_at ?? thread.found_at
  if (!dateStr) return false
  return Date.now() - new Date(dateStr).getTime() > 12 * 60 * 60 * 1000
}

function relevanceBadgeClass(score: number): string {
  if (score >= 70) return 'bg-green-500/20 text-green-400 border-green-500/30'
  if (score >= 40) return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
  return 'bg-red-500/20 text-red-400 border-red-500/30'
}

function SkeletonCard() {
  return (
    <div className="rounded-xl border border-border bg-card p-4 animate-pulse">
      <div className="flex gap-2 mb-3">
        <div className="h-5 w-20 rounded-full bg-muted" />
        <div className="h-5 w-12 rounded-full bg-muted" />
      </div>
      <div className="h-4 w-3/4 rounded bg-muted mb-2" />
      <div className="h-3 w-1/2 rounded bg-muted" />
    </div>
  )
}

const ALL_TABS = ['all', 'new', 'reviewed', 'replied', 'dismissed'] as const
type TabValue = typeof ALL_TABS[number]

export default function Inbox() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [threads, setThreads] = useState<Thread[]>([])
  const [statuses, setStatuses] = useState<StatusRecord[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<TabValue>('all')
  const [onlyRelevant, setOnlyRelevant] = useState(false)
  const [minScore] = useState(50)

  const touchStartX = useRef(0)
  const touchStartY = useRef(0)

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true)
      try {
        const [threadResult, statusResult] = await Promise.all([
          pb.collection('threads').getList<Thread>(1, 100, {
            sort: '-relevance_score',
          }),
          pb.collection('thread_status').getFullList<StatusRecord>({
            filter: `user = "${user?.id ?? ''}"`,
          }),
        ])
        setThreads(threadResult.items)
        setStatuses(statusResult)
      } catch {
        setThreads([])
      } finally {
        setIsLoading(false)
      }
    }

    void fetchData()
  }, [user?.id])

  const dismissThread = async (threadId: string) => {
    const existing = statuses.find(s => s.thread === threadId)
    if (existing) {
      await pb.collection('thread_status').update(existing.id, { status: 'dismissed' })
    }
    setStatuses(prev => prev.map(s => s.thread === threadId ? { ...s, status: 'dismissed' as ThreadStatus } : s))
  }

  const statusMap = new Map<string, ThreadStatus>()
  for (const s of statuses) {
    statusMap.set(s.thread, s.status)
  }

  const filteredThreads = threads.filter((thread) => {
    if (onlyRelevant && thread.relevance_score < minScore) return false
    const status = statusMap.get(thread.id) ?? 'new'
    if (activeTab === 'all') return true
    return status === activeTab
  })

  return (
    <div className="flex flex-col h-full">
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 pt-4 pb-0">
        <div className="flex items-center justify-between mb-3">
          <h1 className="text-xl font-bold">Inbox</h1>
          <div className="flex items-center gap-2">
            <Label htmlFor="only-relevant" className="text-xs text-muted-foreground cursor-pointer">
              Only relevant
            </Label>
            <Switch
              id="only-relevant"
              checked={onlyRelevant}
              onCheckedChange={setOnlyRelevant}
              size="sm"
            />
          </div>
        </div>

        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as TabValue)}>
          <TabsList className="w-full justify-start gap-0 h-auto bg-transparent p-0 border-b-0 rounded-none">
            {ALL_TABS.map((tab) => (
              <TabsTrigger
                key={tab}
                value={tab}
                className="capitalize text-xs px-3 py-2 rounded-none border-b-2 border-transparent data-active:border-primary data-active:bg-transparent"
              >
                {tab}
              </TabsTrigger>
            ))}
          </TabsList>

          {ALL_TABS.map((tab) => (
            <TabsContent key={tab} value={tab} />
          ))}
        </Tabs>
      </div>

      <div className="flex-1 overflow-auto p-4 flex flex-col gap-3">
        <p className="text-xs text-muted-foreground md:hidden">
          Swipe left to dismiss · Swipe right to open
        </p>
        {isLoading ? (
          Array.from({ length: 5 }).map((_, i) => <SkeletonCard key={i} />)
        ) : filteredThreads.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <MessageSquare className="size-10 text-muted-foreground/40 mb-3" />
            <p className="text-sm text-muted-foreground">No threads here yet</p>
          </div>
        ) : (
          filteredThreads.map((thread) => (
            <Card
              key={thread.id}
              className={cn(
                'cursor-pointer hover:bg-muted/30 transition-colors',
                isStale(thread) && 'opacity-50'
              )}
              onClick={() => void navigate(`/threads/${thread.id}`)}
              onTouchStart={(e) => {
                touchStartX.current = e.touches[0].clientX
                touchStartY.current = e.touches[0].clientY
              }}
              onTouchEnd={(e) => {
                const deltaX = e.changedTouches[0].clientX - touchStartX.current
                const deltaY = e.changedTouches[0].clientY - touchStartY.current
                if (Math.abs(deltaX) > 80 && Math.abs(deltaX) > Math.abs(deltaY) * 2) {
                  if (deltaX < 0) {
                    void dismissThread(thread.id)
                  } else {
                    void navigate(`/threads/${thread.id}`)
                  }
                }
              }}
            >
              <CardContent className="p-4">
                <div className="flex items-start justify-between gap-2 mb-2">
                  <div className="flex flex-wrap gap-1.5">
                    <Badge variant="secondary" className="text-xs">
                      r/{thread.subreddit}
                    </Badge>
                    {statusMap.has(thread.id) && (
                      <Badge variant="outline" className="text-xs capitalize">
                        {statusMap.get(thread.id)}
                      </Badge>
                    )}
                  </div>
                  <span
                    className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium shrink-0 ${relevanceBadgeClass(thread.relevance_score)}`}
                  >
                    {thread.relevance_score}%
                  </span>
                </div>

                <h3 className="text-sm font-semibold leading-snug line-clamp-2 mb-2">
                  {thread.title}
                </h3>

                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <ArrowUp className="size-3" />
                    {thread.reddit_score}
                  </span>
                  <span className="flex items-center gap-1">
                    <MessageSquare className="size-3" />
                    {thread.comment_count}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock className="size-3" />
                    {timeAgo(thread.reddit_created_at ?? thread.found_at)}
                  </span>
                  {isStale(thread) && (
                    <span className="flex items-center gap-1 text-yellow-500">
                      <AlertTriangle className="size-3" />
                      Stale
                    </span>
                  )}
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  )
}
