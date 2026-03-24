import { cn } from '@/lib/utils'

export interface Comment {
  id: string
  author: string
  body: string
  score: number
  depth: number
  replies?: Comment[]
}

interface Props {
  comments: Comment[]
  selectedId: string | null
  onSelect: (id: string) => void
  depth?: number
}

export default function CommentTree({ comments, selectedId, onSelect, depth = 0 }: Props) {
  return (
    <div className={cn('flex flex-col gap-2', depth > 0 && 'ml-4 border-l border-border pl-3')}>
      {comments.map((comment) => (
        <div key={comment.id}>
          <button
            type="button"
            onClick={() => onSelect(comment.id)}
            className={cn(
              'w-full cursor-pointer rounded-md p-3 text-left transition-colors hover:bg-muted/50',
              selectedId === comment.id && 'ring-2 ring-primary bg-muted/30'
            )}
          >
            <div className="flex items-center gap-2 mb-1">
              <span className="text-sm font-semibold text-foreground">{comment.author}</span>
              <span className="text-xs text-muted-foreground">
                {comment.score >= 0 ? '+' : ''}{comment.score}
              </span>
            </div>
            <p className="text-sm text-foreground/90 whitespace-pre-wrap">{comment.body}</p>
          </button>
          {comment.replies && comment.replies.length > 0 && (
            <CommentTree
              comments={comment.replies}
              selectedId={selectedId}
              onSelect={onSelect}
              depth={depth + 1}
            />
          )}
        </div>
      ))}
    </div>
  )
}
