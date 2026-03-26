import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import type { RecordModel } from 'pocketbase'
import pb from '@/lib/pocketbase'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Trash2, Plus } from 'lucide-react'

interface Keyword extends RecordModel {
  keyword: string
  subreddits: string[]
  is_active: boolean
  created_by: string
}

export default function Keywords() {
  const { user } = useAuth()
  const [keywords, setKeywords] = useState<Keyword[]>([])
  const [threadCounts, setThreadCounts] = useState<Map<string, number>>(new Map())
  const [isLoading, setIsLoading] = useState(true)
  const [newKeyword, setNewKeyword] = useState('')
  const [newSubreddits, setNewSubreddits] = useState('')
  const [isAdding, setIsAdding] = useState(false)

  const fetchKeywords = async () => {
    try {
      const result = await pb.collection('keywords').getFullList<Keyword>({
        filter: `created_by = "${user?.id ?? ''}"`,
        sort: '-created',
      })
      setKeywords(result)

      const allThreads = await pb.collection('threads').getFullList({
        fields: 'id,matched_keyword',
      })
      const counts = new Map<string, number>()
      for (const t of allThreads) {
        const kwId = t.matched_keyword as string
        if (kwId) {
          counts.set(kwId, (counts.get(kwId) ?? 0) + 1)
        }
      }
      setThreadCounts(counts)
    } catch (_e) {
      void _e
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    void fetchKeywords()
  }, [user?.id])

  const handleAdd = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!newKeyword.trim()) return
    setIsAdding(true)
    try {
      const subreddits = newSubreddits
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      const created = await pb.collection('keywords').create<Keyword>({
        keyword: newKeyword.trim(),
        subreddits,
        is_active: true,
        created_by: user?.id,
      })
      setKeywords((prev) => [created, ...prev])
      setNewKeyword('')
      setNewSubreddits('')
    } catch (_e) {
      void _e
    } finally {
      setIsAdding(false)
    }
  }

  const handleToggle = async (id: string, isActive: boolean) => {
    setKeywords((prev) =>
      prev.map((kw) => (kw.id === id ? { ...kw, is_active: isActive } : kw))
    )
    try {
      await pb.collection('keywords').update(id, { is_active: isActive })
    } catch (_e) {
      void _e
      setKeywords((prev) =>
        prev.map((kw) => (kw.id === id ? { ...kw, is_active: !isActive } : kw))
      )
    }
  }

  const handleDelete = async (id: string) => {
    setKeywords((prev) => prev.filter((kw) => kw.id !== id))
    try {
      await pb.collection('keywords').delete(id)
    } catch (_e) {
      void _e
      void fetchKeywords()
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-4">
        <h1 className="text-xl font-bold">Keywords</h1>
      </div>

      <div className="flex-1 overflow-auto p-4 flex flex-col gap-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <h2 className="text-sm font-semibold mb-3">Add Keyword</h2>
          <form onSubmit={(e) => void handleAdd(e)} className="flex flex-col gap-3">
            <Input
              placeholder="Keyword to monitor…"
              value={newKeyword}
              onChange={(e) => setNewKeyword(e.target.value)}
              required
            />
            <Input
              placeholder="Subreddits (comma-separated, optional)"
              value={newSubreddits}
              onChange={(e) => setNewSubreddits(e.target.value)}
            />
            <Button type="submit" disabled={isAdding} className="gap-2 self-start">
              <Plus className="size-4" />
              {isAdding ? 'Adding…' : 'Add Keyword'}
            </Button>
          </form>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-10">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-muted border-t-primary" />
          </div>
        ) : keywords.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <p className="text-sm text-muted-foreground">No keywords yet. Add one above.</p>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {keywords.map((kw) => (
              <div
                key={kw.id}
                className="flex items-center gap-3 rounded-xl border border-border bg-card px-4 py-3"
              >
                <Switch
                  checked={kw.is_active}
                  onCheckedChange={(checked) => void handleToggle(kw.id, checked)}
                  size="sm"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{kw.keyword}</p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {threadCounts.get(kw.id) ?? 0} threads found
                  </p>
                  {kw.subreddits.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-1">
                      {kw.subreddits.map((sub) => (
                        <Badge key={sub} variant="secondary" className="text-xs">
                          r/{sub}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => void handleDelete(kw.id)}
                  className="text-muted-foreground hover:text-destructive shrink-0"
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
