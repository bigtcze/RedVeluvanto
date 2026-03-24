import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router'
import type { RecordModel } from 'pocketbase'
import pb from '@/lib/pocketbase'
import { useAuth } from '@/lib/auth'
import CommentTree from '@/components/CommentTree'
import type { Comment } from '@/components/CommentTree'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ArrowLeft, ArrowUp, RefreshCw, Send, Save, Wand2, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '@/lib/utils'

interface Thread extends RecordModel {
  title: string
  subreddit: string
  body: string
  author: string
  reddit_score: number
  comment_count: number
  relevance_score: number
  reddit_url: string
  comments_tree: string
  subreddit_rules?: string
}

interface Persona extends RecordModel {
  name: string
}

interface DraftRecord extends RecordModel {
  generated_text: string
}

interface DraftHistoryRecord extends RecordModel {
  generated_text: string
  edited_text?: string
  status: 'draft' | 'posted' | 'failed'
}

function relevanceBadgeClass(score: number): string {
  if (score >= 70) return 'bg-green-500/20 text-green-400 border-green-500/30'
  if (score >= 40) return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
  return 'bg-red-500/20 text-red-400 border-red-500/30'
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

function historyStatusVariant(status: string): 'default' | 'secondary' | 'destructive' {
  if (status === 'posted') return 'default'
  if (status === 'failed') return 'destructive'
  return 'secondary'
}

function detectLanguage(text: string): string {
  const czWords = /\b(je|a|v|na|to|se|že|pro|aby|jak|co|ale|tak|jako|ten|být|mít|který)\b/gi
  const enWords = /\b(the|is|and|in|to|of|a|for|that|with|are|this|was|have|but|not|you|they)\b/gi
  const czCount = (text.match(czWords) ?? []).length
  const enCount = (text.match(enWords) ?? []).length
  if (czCount > enCount && czCount > 3) return 'CS'
  if (enCount > czCount && enCount > 3) return 'EN'
  if (czCount > 0) return 'CS'
  return 'EN'
}

function findCommentBody(comments: Comment[], id: string): string {
  for (const c of comments) {
    if (c.id === id) return c.body
    if (c.replies) {
      const found = findCommentBody(c.replies, id)
      if (found !== '') return found
    }
  }
  return ''
}

export default function ThreadDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [thread, setThread] = useState<Thread | null>(null)
  const [personas, setPersonas] = useState<Persona[]>([])
  const [comments, setComments] = useState<Comment[]>([])
  const [selectedCommentId, setSelectedCommentId] = useState<string | null>(null)
  const [selectedPersonaId, setSelectedPersonaId] = useState<string>('')
  const [draftId, setDraftId] = useState<string | null>(null)
  const [draftContent, setDraftContent] = useState('')
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [isGenerating, setIsGenerating] = useState(false)
  const [isApproving, setIsApproving] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [rulesExpanded, setRulesExpanded] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [draftHistory, setDraftHistory] = useState<DraftHistoryRecord[]>([])
  const [historyExpanded, setHistoryExpanded] = useState(false)

  useEffect(() => {
    const fetchData = async () => {
      if (!id) return
      setIsLoading(true)
      try {
        const [threadData, personasData] = await Promise.all([
          pb.collection('threads').getOne<Thread>(id),
          pb.collection('personas').getFullList<Persona>(),
        ])
        setThread(threadData)
        setPersonas(personasData)
        if (personasData.length > 0) {
          setSelectedPersonaId(personasData[0].id)
        }
        if (threadData.comments_tree) {
          try {
            setComments(JSON.parse(threadData.comments_tree) as Comment[])
          } catch {
            setComments([])
          }
        }
      } catch {
        setThread(null)
      } finally {
        setIsLoading(false)
      }
    }
    void fetchData()
  }, [id])

  useEffect(() => {
    const fetchHistory = async () => {
      if (!id || !user?.id) return
      try {
        const history = await pb.collection('drafts').getFullList<DraftHistoryRecord>({
          filter: `thread = "${id}" && user = "${user.id}"`,
          sort: '-created',
        })
        setDraftHistory(history)
      } catch {
        setDraftHistory([])
      }
    }
    void fetchHistory()
  }, [id, user?.id, draftId])

  const handleRefreshComments = async () => {
    if (!id) return
    setIsRefreshing(true)
    try {
      const res = await fetch(`/api/threads/${id}/refresh`, {
        method: 'POST',
        headers: { Authorization: pb.authStore.token },
      })
      if (res.ok) {
        const updated = (await res.json()) as Thread
        setThread(updated)
        if (updated.comments_tree) {
          try {
            setComments(JSON.parse(updated.comments_tree) as Comment[])
          } catch {
            setComments([])
          }
        }
      }
    } catch (_e) {
      void _e
    } finally {
      setIsRefreshing(false)
    }
  }

  const handleGenerate = async () => {
    if (!id) return
    setIsGenerating(true)
    try {
      const res = await fetch('/api/drafts/generate', {
        method: 'POST',
        headers: {
          Authorization: pb.authStore.token,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          thread_id: id,
          persona_id: selectedPersonaId,
          parent_comment_id: selectedCommentId,
        }),
      })
      const data = (await res.json()) as DraftRecord
      setDraftId(data.id)
      setDraftContent(data.generated_text ?? '')
    } catch (_e) {
      void _e
    } finally {
      setIsGenerating(false)
    }
  }

  const handleRegenerate = async () => {
    if (!draftId) return
    setIsGenerating(true)
    try {
      const res = await fetch(`/api/drafts/${draftId}/regenerate`, {
        method: 'POST',
        headers: { Authorization: pb.authStore.token },
      })
      const data = (await res.json()) as DraftRecord
      setDraftContent(data.generated_text ?? '')
    } catch (_e) {
      void _e
    } finally {
      setIsGenerating(false)
    }
  }

  const handleApprove = async () => {
    if (!draftId) return
    setIsApproving(true)
    try {
      await fetch(`/api/drafts/${draftId}/approve`, {
        method: 'POST',
        headers: { Authorization: pb.authStore.token },
      })
    } catch (_e) {
      void _e
    } finally {
      setIsApproving(false)
    }
  }

  const handleSaveDraft = async () => {
    if (!draftId) return
    setIsSaving(true)
    try {
      await pb.collection('drafts').update(draftId, { edited_text: draftContent })
    } catch (_e) {
      void _e
    } finally {
      setIsSaving(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-muted border-t-primary" />
      </div>
    )
  }

  if (!thread) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-muted-foreground">Thread not found.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col md:flex-row h-full">
      <div className="flex-1 overflow-auto p-4 flex flex-col gap-4 md:pr-2">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon-sm" onClick={() => void navigate(-1)}>
            <ArrowLeft className="size-4" />
          </Button>
          <Badge variant="secondary">r/{thread.subreddit}</Badge>
          <span
            className={cn(
              'inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium',
              relevanceBadgeClass(thread.relevance_score)
            )}
          >
            {thread.relevance_score}%
          </span>
        </div>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base leading-snug">{thread.title}</CardTitle>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span>u/{thread.author}</span>
              <span className="flex items-center gap-0.5">
                <ArrowUp className="size-3" />
                {thread.reddit_score}
              </span>
              {thread.reddit_url && (
                <a
                  href={thread.reddit_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="ml-auto text-primary hover:underline"
                >
                  View on Reddit ↗
                </a>
              )}
            </div>
          </CardHeader>
          {thread.body && (
            <CardContent>
              <p className="text-sm text-foreground/80 whitespace-pre-wrap leading-relaxed">
                {thread.body}
              </p>
            </CardContent>
          )}
        </Card>

        {thread.subreddit_rules && (
          <div className="rounded-xl border border-border bg-card">
            <button
              type="button"
              onClick={() => setRulesExpanded((v) => !v)}
              className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium"
            >
              <span>Subreddit Rules</span>
              {rulesExpanded ? (
                <ChevronUp className="size-4 text-muted-foreground" />
              ) : (
                <ChevronDown className="size-4 text-muted-foreground" />
              )}
            </button>
            {rulesExpanded && (
              <div className="border-t border-border px-4 py-3">
                <p className="text-xs text-muted-foreground whitespace-pre-wrap leading-relaxed">
                  {thread.subreddit_rules}
                </p>
              </div>
            )}
          </div>
        )}

        <div>
          <div className="flex items-center justify-between mb-2">
            <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              Comments ({thread.comment_count})
            </h2>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => void handleRefreshComments()}
              disabled={isRefreshing}
              className="text-muted-foreground hover:text-foreground"
            >
              <RefreshCw className={cn('size-3.5', isRefreshing && 'animate-spin')} />
            </Button>
          </div>
          {comments.length > 0 ? (
            <CommentTree
              comments={comments}
              selectedId={selectedCommentId}
              onSelect={(cid) =>
                setSelectedCommentId((prev) => (prev === cid ? null : cid))
              }
            />
          ) : (
            <p className="text-sm text-muted-foreground py-4 text-center">No comments loaded.</p>
          )}
        </div>
      </div>

      <aside className="md:w-80 md:shrink-0 border-t md:border-t-0 md:border-l border-border bg-card/50 p-4 flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          Reply
        </h2>

        {selectedCommentId && (() => {
          const body = findCommentBody(comments, selectedCommentId)
          const lang = detectLanguage(body)
          const flag = lang === 'CS' ? '🇨🇿' : '🇬🇧'
          return (
            <div className="rounded-md border border-primary/30 bg-primary/5 px-3 py-2 text-xs text-muted-foreground">
              Replying to selected comment · {flag} {lang}
            </div>
          )
        })()}

        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-muted-foreground">Persona</label>
            <Select value={selectedPersonaId} onValueChange={(v) => { if (v !== null) setSelectedPersonaId(v) }}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Select persona…" />
            </SelectTrigger>
            <SelectContent>
              {personas.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <Button
          onClick={() => void handleGenerate()}
          disabled={isGenerating || !selectedPersonaId}
          className="w-full gap-2"
        >
          <Wand2 className="size-4" />
          {isGenerating ? 'Generating…' : 'Generate Reply'}
        </Button>

        {draftContent && (
          <>
            <Textarea
              value={draftContent}
              onChange={(e) => setDraftContent(e.target.value)}
              className="min-h-36 resize-y text-sm"
              placeholder="Generated reply will appear here…"
            />
            <div className="flex flex-col gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleRegenerate()}
                disabled={isGenerating}
                className="w-full gap-2"
              >
                <RefreshCw className="size-3.5" />
                Try Again
              </Button>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void handleSaveDraft()}
                  disabled={isSaving}
                  className="flex-1 gap-1.5"
                >
                  <Save className="size-3.5" />
                  {isSaving ? 'Saving…' : 'Save Draft'}
                </Button>
                <Button
                  size="sm"
                  onClick={() => void handleApprove()}
                  disabled={isApproving}
                  className="flex-1 gap-1.5"
                >
                  <Send className="size-3.5" />
                  {isApproving ? 'Sending…' : 'Approve & Send'}
                </Button>
              </div>
            </div>
          </>
        )}

        {!draftContent && (
          <Textarea
            value={draftContent}
            onChange={(e) => setDraftContent(e.target.value)}
            className="min-h-36 resize-y text-sm"
            placeholder="Generated reply will appear here…"
          />
        )}

        {draftHistory.length > 0 && (
          <div className="rounded-xl border border-border bg-card/30">
            <button
              type="button"
              onClick={() => setHistoryExpanded((v) => !v)}
              className="flex w-full items-center justify-between px-3 py-2.5 text-xs font-medium"
            >
              <span className="text-muted-foreground uppercase tracking-wide">
                Draft History ({draftHistory.length})
              </span>
              {historyExpanded ? (
                <ChevronUp className="size-3.5 text-muted-foreground" />
              ) : (
                <ChevronDown className="size-3.5 text-muted-foreground" />
              )}
            </button>
            {historyExpanded && (
              <div className="border-t border-border flex flex-col divide-y divide-border">
                {draftHistory.map((draft) => (
                  <button
                    key={draft.id}
                    type="button"
                    className="flex flex-col gap-1 px-3 py-2.5 text-left hover:bg-muted/30 transition-colors"
                    onClick={() => {
                      setDraftContent(draft.edited_text ?? draft.generated_text)
                      setDraftId(draft.id)
                    }}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <Badge variant={historyStatusVariant(draft.status)} className="text-xs capitalize">
                        {draft.status}
                      </Badge>
                      <span className="text-xs text-muted-foreground">{timeAgo(draft.created)}</span>
                    </div>
                    <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed">
                      {(draft.edited_text ?? draft.generated_text).slice(0, 100)}
                    </p>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </aside>
    </div>
  )
}
