import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import type { RecordModel } from 'pocketbase'
import pb from '@/lib/pocketbase'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import {
  ArrowLeft,
  Plus,
  Trash2,
  Copy,
  Loader2,
  ChevronDown,
  ChevronUp,
  X,
} from 'lucide-react'

// ─── Types ────────────────────────────────────────────────────────────────────

type ReplyGoal = 'help' | 'promote' | 'reputation' | 'traffic' | 'educate'
type CompetitorStance = 'ignore' | 'acknowledge' | 'compare_fairly' | 'differentiate'

interface PersonaTraits {
  formality: number
  verbosity: number
  humor: number
  empathy: number
  confidence: number
  expertise: number
  controversy: number
  emoji_usage: number
  typo_tolerance: number
}

interface PersonaRecord extends RecordModel {
  name: string
  description: string
  traits: PersonaTraits
  custom_traits: string[]
  reply_goal: ReplyGoal
  reply_goal_detail: string
  behavior_rules: string[]
  competitor_stance: CompetitorStance
  competitor_names: string[]
  forbidden_words: string[]
  max_length: number
  language: string
  knowledge_text: string
  knowledge_urls: string[]
  examples: string[]
  is_default: boolean
  created_by: string
}

interface PreviewResponse {
  preview: string
  system_prompt: string
}

// ─── Constants ────────────────────────────────────────────────────────────────

const DEFAULT_TRAITS: PersonaTraits = {
  formality: 5,
  verbosity: 5,
  humor: 3,
  empathy: 5,
  confidence: 6,
  expertise: 7,
  controversy: 4,
  emoji_usage: 2,
  typo_tolerance: 2,
}

const TRAIT_DEFS: Array<{ key: keyof PersonaTraits; label: string; left: string; right: string }> = [
  { key: 'formality', label: 'Formality', left: 'Informal', right: 'Formal' },
  { key: 'verbosity', label: 'Verbosity', left: 'Brief', right: 'Detailed' },
  { key: 'humor', label: 'Humor', left: 'Serious', right: 'Sarcastic' },
  { key: 'empathy', label: 'Empathy', left: 'Factual', right: 'Empathetic' },
  { key: 'confidence', label: 'Confidence', left: 'Cautious', right: 'Confident' },
  { key: 'expertise', label: 'Expertise', left: 'Learner', right: 'Expert' },
  { key: 'controversy', label: 'Controversy', left: 'Agreeable', right: 'Challenging' },
  { key: 'emoji_usage', label: 'Emoji Usage', left: 'None', right: 'Heavy' },
  { key: 'typo_tolerance', label: 'Typo Tolerance', left: 'Perfect', right: 'Casual' },
]

const REPLY_GOAL_OPTS: Array<{ value: ReplyGoal; label: string; description: string }> = [
  { value: 'help', label: 'Help', description: "Purely help, don't mention product unless directly relevant" },
  { value: 'promote', label: 'Promote', description: 'Look for natural product mention opportunities' },
  { value: 'reputation', label: 'Reputation', description: 'Build expert reputation, product only peripherally' },
  { value: 'traffic', label: 'Traffic', description: 'Guide readers to website/link naturally' },
  { value: 'educate', label: 'Educate', description: 'Share knowledge, product as example' },
]

const COMPETITOR_STANCE_OPTS: Array<{ value: CompetitorStance; label: string; description: string }> = [
  { value: 'ignore', label: 'Ignore', description: 'Never mention competitors' },
  { value: 'acknowledge', label: 'Acknowledge', description: 'Briefly acknowledge existence' },
  { value: 'compare_fairly', label: 'Compare Fairly', description: 'Compare pros and cons fairly' },
  { value: 'differentiate', label: 'Differentiate', description: 'Emphasize differences without criticism' },
]

const LANGUAGE_OPTS = [
  { value: 'en', label: 'English' },
  { value: 'cs', label: 'Czech' },
  { value: 'de', label: 'German' },
  { value: 'es', label: 'Spanish' },
  { value: 'fr', label: 'French' },
  { value: 'other', label: 'Other' },
]

// ─── Sub-components ───────────────────────────────────────────────────────────

function SectionHeader({ title, description }: { title: string; description: string }) {
  return (
    <div className="mb-4">
      <h2 className="text-sm font-semibold">{title}</h2>
      <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
    </div>
  )
}

// ─── Main Component ───────────────────────────────────────────────────────────

export default function PersonaBuilder() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()

  const isEditMode = Boolean(id)

  // ── Form state ──────────────────────────────────────────────────────────────
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [traits, setTraits] = useState<PersonaTraits>({ ...DEFAULT_TRAITS })
  const [customTraitsText, setCustomTraitsText] = useState('')
  const [replyGoal, setReplyGoal] = useState<ReplyGoal>('help')
  const [replyGoalDetail, setReplyGoalDetail] = useState('')
  const [behaviorRules, setBehaviorRules] = useState<string[]>([])
  const [newRule, setNewRule] = useState('')
  const [competitorStance, setCompetitorStance] = useState<CompetitorStance>('ignore')
  const [competitorNames, setCompetitorNames] = useState<string[]>([])
  const [newCompetitorName, setNewCompetitorName] = useState('')
  const [knowledgeText, setKnowledgeText] = useState('')
  const [knowledgeUrls, setKnowledgeUrls] = useState<string[]>([])
  const [knowledgeTab, setKnowledgeTab] = useState('text')
  const [newUrl, setNewUrl] = useState('')
  const [forbiddenWords, setForbiddenWords] = useState<string[]>([])
  const [newForbiddenWord, setNewForbiddenWord] = useState('')
  const [maxLength, setMaxLength] = useState(0)
  const [language, setLanguage] = useState('en')
  const [examples, setExamples] = useState<string[]>([])
  const [isDefault, setIsDefault] = useState(false)

  // ── Preview state ───────────────────────────────────────────────────────────
  const [previewText, setPreviewText] = useState('')
  const [systemPrompt, setSystemPrompt] = useState('')
  const [isGeneratingPreview, setIsGeneratingPreview] = useState(false)
  const [showSystemPrompt, setShowSystemPrompt] = useState(false)
  const [showMobilePreview, setShowMobilePreview] = useState(false)

  // ── UI state ────────────────────────────────────────────────────────────────
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [isLoading, setIsLoading] = useState(isEditMode)

  // ── Load existing persona ───────────────────────────────────────────────────
  useEffect(() => {
    if (!id) return
    const load = async () => {
      try {
        const r = await pb.collection('personas').getOne<PersonaRecord>(id)
        setName(r.name ?? '')
        setDescription(r.description ?? '')
        if (r.traits) setTraits(r.traits)
        if (Array.isArray(r.custom_traits)) setCustomTraitsText((r.custom_traits as string[]).join('\n'))
        if (r.reply_goal) setReplyGoal(r.reply_goal)
        setReplyGoalDetail(r.reply_goal_detail ?? '')
        if (Array.isArray(r.behavior_rules)) setBehaviorRules(r.behavior_rules as string[])
        if (r.competitor_stance) setCompetitorStance(r.competitor_stance)
        if (Array.isArray(r.competitor_names)) setCompetitorNames(r.competitor_names as string[])
        if (Array.isArray(r.forbidden_words)) setForbiddenWords(r.forbidden_words as string[])
        setMaxLength(r.max_length ?? 0)
        setLanguage(r.language ?? 'en')
        setKnowledgeText(r.knowledge_text ?? '')
        if (Array.isArray(r.knowledge_urls)) setKnowledgeUrls(r.knowledge_urls as string[])
        if (Array.isArray(r.examples)) setExamples(r.examples as string[])
        setIsDefault(r.is_default ?? false)
      } catch (_e) {
        void _e
      } finally {
        setIsLoading(false)
      }
    }
    void load()
  }, [id])

  // ── Helpers ─────────────────────────────────────────────────────────────────
  const buildData = () => ({
    name,
    description,
    traits,
    custom_traits: customTraitsText.split('\n').map((s) => s.trim()).filter(Boolean),
    reply_goal: replyGoal,
    reply_goal_detail: replyGoalDetail,
    behavior_rules: behaviorRules,
    competitor_stance: competitorStance,
    competitor_names: competitorNames,
    forbidden_words: forbiddenWords,
    max_length: maxLength,
    language,
    knowledge_text: knowledgeText,
    knowledge_urls: knowledgeUrls,
    examples,
    is_default: isDefault,
  })

  // ── Actions ─────────────────────────────────────────────────────────────────
  const handleSave = async () => {
    if (!name.trim()) return
    setIsSaving(true)
    try {
      const data = buildData()
      if (isEditMode && id) {
        await pb.collection('personas').update(id, data)
      } else {
        await pb.collection('personas').create({ ...data, created_by: user?.id })
      }
      navigate('/settings')
    } catch (_e) {
      void _e
    } finally {
      setIsSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!id) return
    setIsDeleting(true)
    try {
      await pb.collection('personas').delete(id)
      navigate('/settings')
    } catch (_e) {
      void _e
    } finally {
      setIsDeleting(false)
      setShowDeleteDialog(false)
    }
  }

  const handleDuplicate = async () => {
    setIsSaving(true)
    try {
      const data = { ...buildData(), name: `Copy of ${name}` }
      const created = await pb.collection('personas').create({ ...data, created_by: user?.id })
      navigate(`/personas/${created.id}`)
    } catch (_e) {
      void _e
    } finally {
      setIsSaving(false)
    }
  }

  const handleGeneratePreview = async () => {
    setIsGeneratingPreview(true)
    try {
      const res = await fetch('/api/personas/preview', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: pb.authStore.token,
        },
        body: JSON.stringify(buildData()),
      })
      const result = (await res.json()) as PreviewResponse
      setPreviewText(result.preview ?? '')
      setSystemPrompt(result.system_prompt ?? '')
    } catch (_e) {
      void _e
    } finally {
      setIsGeneratingPreview(false)
    }
  }

  const addRule = () => {
    const val = newRule.trim()
    if (!val) return
    setBehaviorRules((prev) => [...prev, val])
    setNewRule('')
  }

  const addCompetitorName = () => {
    const val = newCompetitorName.trim()
    if (val && !competitorNames.includes(val)) {
      setCompetitorNames((prev) => [...prev, val])
    }
    setNewCompetitorName('')
  }

  const addForbiddenWord = () => {
    const val = newForbiddenWord.trim()
    if (val && !forbiddenWords.includes(val)) {
      setForbiddenWords((prev) => [...prev, val])
    }
    setNewForbiddenWord('')
  }

  const addUrl = () => {
    const val = newUrl.trim()
    if (!val) return
    setKnowledgeUrls((prev) => [...prev, val])
    setNewUrl('')
  }

  // ── Loading state ───────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const replyGoalDescription = REPLY_GOAL_OPTS.find((o) => o.value === replyGoal)?.description

  // ── Preview panel (shared between desktop sidebar and mobile collapse) ──────
  const previewPanel = (
    <div className="flex flex-col gap-4">
      <div>
        <h3 className="text-sm font-semibold mb-1">Live Preview</h3>
        <p className="text-xs text-muted-foreground">
          Generate a sample response using current persona settings.
        </p>
      </div>
      <Button
        size="sm"
        variant="outline"
        onClick={() => void handleGeneratePreview()}
        disabled={isGeneratingPreview}
        className="gap-2 self-start"
      >
        {isGeneratingPreview && <Loader2 className="size-3.5 animate-spin" />}
        {isGeneratingPreview ? 'Generating…' : 'Generate Preview'}
      </Button>

      {previewText && (
        <div className="rounded-lg border border-border bg-muted/30 p-3">
          <p className="text-sm whitespace-pre-wrap leading-relaxed">{previewText}</p>
        </div>
      )}

      {systemPrompt && (
        <div>
          <button
            type="button"
            onClick={() => setShowSystemPrompt((s) => !s)}
            className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {showSystemPrompt ? (
              <ChevronUp className="size-3" />
            ) : (
              <ChevronDown className="size-3" />
            )}
            View System Prompt
          </button>
          {showSystemPrompt && (
            <pre className="mt-2 rounded-lg border border-border bg-muted/50 p-3 text-xs overflow-auto max-h-60 whitespace-pre-wrap leading-relaxed">
              {systemPrompt}
            </pre>
          )}
        </div>
      )}
    </div>
  )

  // ── Render ──────────────────────────────────────────────────────────────────
  return (
    <div className="flex flex-col h-full">
      {/* ── Sticky header ── */}
      <div className="sticky top-0 z-20 bg-background/95 backdrop-blur border-b border-border px-4 py-3 flex items-center justify-between gap-3 shrink-0">
        <div className="flex items-center gap-2 min-w-0">
          <Button variant="ghost" size="icon-sm" onClick={() => navigate('/settings')}>
            <ArrowLeft className="size-4" />
          </Button>
          <h1 className="text-base font-bold truncate">
            {isEditMode ? 'Edit Persona' : 'New Persona'}
          </h1>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {isEditMode && (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => void handleDuplicate()}
                disabled={isSaving}
                className="hidden sm:flex gap-1.5"
              >
                <Copy className="size-3.5" />
                Duplicate
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setShowDeleteDialog(true)}
                className="hidden sm:flex gap-1.5"
              >
                <Trash2 className="size-3.5" />
                Delete
              </Button>
            </>
          )}
          <Button
            size="sm"
            onClick={() => void handleSave()}
            disabled={isSaving || !name.trim()}
            className="gap-1.5"
          >
            {isSaving && <Loader2 className="size-3.5 animate-spin" />}
            {isSaving ? 'Saving…' : 'Save'}
          </Button>
        </div>
      </div>

      {/* ── Two-column layout ── */}
      <div className="flex flex-1 min-h-0">
        {/* Form column */}
        <div className="flex-1 overflow-y-auto p-4 md:p-6 flex flex-col gap-5 pb-4">

          {/* ── 1. Basic Info ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Basic Info"
              description="Name and description for your own reference."
            />
            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium">
                  Name <span className="text-destructive">*</span>
                </label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. Expert Advisor, Friendly Helper"
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium">Description</label>
                <Textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Short note for your own reference…"
                  className="min-h-[72px]"
                />
              </div>
            </div>
          </section>

          {/* ── 2. Personality Traits ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Personality Traits"
              description="Adjust sliders to define the AI's communication style."
            />
            <div className="flex flex-col gap-5">
              {TRAIT_DEFS.map((t) => (
                <div key={t.key}>
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-sm font-medium">{t.label}</label>
                    <span className="text-sm font-mono tabular-nums text-primary w-5 text-right">
                      {traits[t.key]}
                    </span>
                  </div>
                  <Slider
                    min={0}
                    max={10}
                    step={1}
                    value={[traits[t.key]]}
                    onValueChange={(val) => {
                      const v = Array.isArray(val) ? val[0] : val
                      if (typeof v === 'number') setTraits((prev) => ({ ...prev, [t.key]: v }))
                    }}
                  />
                  <div className="flex justify-between mt-1.5 text-xs text-muted-foreground">
                    <span>{t.left}</span>
                    <span>{t.right}</span>
                  </div>
                </div>
              ))}
            </div>
          </section>

          {/* ── 3. Custom Instructions ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Custom Instructions"
              description="Free-form instructions for the AI, one per line."
            />
            <Textarea
              value={customTraitsText}
              onChange={(e) => setCustomTraitsText(e.target.value)}
              placeholder={
                'Uses Czech words in English text\nAvoids technical jargon\nAlways asks a follow-up question'
              }
              className="min-h-[96px] font-mono text-sm"
            />
            <p className="mt-2 text-xs text-muted-foreground">
              E.g.: "Uses Czech words in English text"
            </p>
          </section>

          {/* ── 4. Reply Goal ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Reply Goal"
              description="What should the persona try to achieve with each reply?"
            />
            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium">Goal</label>
                <Select
                  value={replyGoal}
                  onValueChange={(val) => {
                    if (val) setReplyGoal(val as ReplyGoal)
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {REPLY_GOAL_OPTS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {replyGoalDescription && (
                  <p className="text-xs text-muted-foreground">{replyGoalDescription}</p>
                )}
              </div>
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium">
                  Additional Details{' '}
                  <span className="font-normal text-muted-foreground">(optional)</span>
                </label>
                <Textarea
                  value={replyGoalDetail}
                  onChange={(e) => setReplyGoalDetail(e.target.value)}
                  placeholder="Specify when/how to mention product…"
                  className="min-h-[72px]"
                />
              </div>
            </div>
          </section>

          {/* ── 5. Behavior Rules ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Behavior Rules"
              description="Hard rules the AI must always follow."
            />
            <div className="flex flex-col gap-2">
              {behaviorRules.length > 0 && (
                <div className="flex flex-col gap-1.5 mb-1">
                  {behaviorRules.map((rule, idx) => (
                    <div
                      key={idx}
                      className="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2"
                    >
                      <p className="flex-1 text-sm">{rule}</p>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        onClick={() =>
                          setBehaviorRules((prev) => prev.filter((_, i) => i !== idx))
                        }
                        className="shrink-0 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
              <div className="flex gap-2">
                <Input
                  value={newRule}
                  onChange={(e) => setNewRule(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      addRule()
                    }
                  }}
                  placeholder="Never promise features that don't exist…"
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addRule}
                  className="gap-1.5 shrink-0"
                >
                  <Plus className="size-3.5" />
                  Add
                </Button>
              </div>
            </div>
          </section>

          {/* ── 6. Competitor Settings ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Competitor Settings"
              description="How should the persona handle competitor mentions?"
            />
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium">Stance</label>
                <Select
                  value={competitorStance}
                  onValueChange={(val) => {
                    if (val) setCompetitorStance(val as CompetitorStance)
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {COMPETITOR_STANCE_OPTS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  {COMPETITOR_STANCE_OPTS.find((o) => o.value === competitorStance)?.description}
                </p>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium">Competitor Names</label>
                {competitorNames.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 mb-1">
                    {competitorNames.map((n) => (
                      <Badge key={n} variant="secondary" className="gap-1 pr-1">
                        {n}
                        <button
                          type="button"
                          onClick={() =>
                            setCompetitorNames((prev) => prev.filter((x) => x !== n))
                          }
                          className="rounded-full hover:bg-foreground/20 p-0.5"
                        >
                          <X className="size-2.5" />
                        </button>
                      </Badge>
                    ))}
                  </div>
                )}
                <Input
                  value={newCompetitorName}
                  onChange={(e) => setNewCompetitorName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      addCompetitorName()
                    }
                  }}
                  placeholder="Type a competitor name and press Enter…"
                />
                <p className="text-xs text-muted-foreground">Press Enter to add</p>
              </div>
            </div>
          </section>

          {/* ── 7. Knowledge Base ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Knowledge Base"
              description="Product information the AI should know."
            />
            <Tabs value={knowledgeTab} onValueChange={(val) => setKnowledgeTab(val)}>
              <TabsList className="mb-3">
                <TabsTrigger value="text">Text</TabsTrigger>
                <TabsTrigger value="urls">URLs</TabsTrigger>
              </TabsList>
              <TabsContent value="text">
                <Textarea
                  value={knowledgeText}
                  onChange={(e) => setKnowledgeText(e.target.value)}
                  placeholder="Describe your product, features, pricing, FAQs…"
                  className="min-h-[160px]"
                />
              </TabsContent>
              <TabsContent value="urls">
                <div className="flex flex-col gap-2">
                  {knowledgeUrls.length > 0 && (
                    <div className="flex flex-col gap-1.5 mb-1">
                      {knowledgeUrls.map((url, idx) => (
                        <div
                          key={idx}
                          className="flex items-center gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2"
                        >
                          <p className="flex-1 text-sm break-all">{url}</p>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon-sm"
                            onClick={() =>
                              setKnowledgeUrls((prev) => prev.filter((_, i) => i !== idx))
                            }
                            className="shrink-0 text-muted-foreground hover:text-destructive"
                          >
                            <Trash2 className="size-3.5" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}
                  <div className="flex gap-2">
                    <Input
                      type="url"
                      value={newUrl}
                      onChange={(e) => setNewUrl(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault()
                          addUrl()
                        }
                      }}
                      placeholder="https://example.com/about"
                      className="flex-1"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={addUrl}
                      className="gap-1.5 shrink-0"
                    >
                      <Plus className="size-3.5" />
                      Add
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    URLs are crawled by the backend when saving.
                  </p>
                </div>
              </TabsContent>
            </Tabs>
          </section>

          {/* ── 8. Forbidden Words ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Forbidden Words"
              description="Words and phrases the AI should never use."
            />
            {forbiddenWords.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mb-3">
                {forbiddenWords.map((word) => (
                  <Badge key={word} variant="destructive" className="gap-1 pr-1">
                    {word}
                    <button
                      type="button"
                      onClick={() => setForbiddenWords((prev) => prev.filter((w) => w !== word))}
                      className="rounded-full hover:bg-foreground/20 p-0.5"
                    >
                      <X className="size-2.5" />
                    </button>
                  </Badge>
                ))}
              </div>
            )}
            <Input
              value={newForbiddenWord}
              onChange={(e) => setNewForbiddenWord(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  addForbiddenWord()
                }
              }}
              placeholder="Type a word and press Enter…"
            />
            <p className="mt-1.5 text-xs text-muted-foreground">Press Enter to add</p>
          </section>

          {/* ── 9. Max Reply Length ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Max Reply Length"
              description="Maximum character count for generated replies."
            />
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium">Limit</label>
                <span className="text-sm font-mono tabular-nums text-primary">
                  {maxLength === 0 ? 'No limit' : `${maxLength} chars`}
                </span>
              </div>
              <Slider
                min={0}
                max={2000}
                step={50}
                value={[maxLength]}
                onValueChange={(val) => {
                  const v = Array.isArray(val) ? val[0] : val
                  if (typeof v === 'number') setMaxLength(v)
                }}
              />
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>No limit</span>
                <span>2000</span>
              </div>
            </div>
          </section>

          {/* ── 10. Default Language ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Default Language"
              description="Fallback language when auto-detection isn't possible."
            />
            <Select value={language} onValueChange={(val) => { if (val) setLanguage(val) }}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {LANGUAGE_OPTS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="mt-2 text-xs text-muted-foreground">
              Auto-detection from comment language always takes priority.
            </p>
          </section>

          {/* ── 11. Example Responses ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <SectionHeader
              title="Example Responses"
              description="Few-shot examples to guide the AI's style."
            />
            <div className="flex flex-col gap-3">
              {examples.map((ex, idx) => (
                <div key={idx} className="flex gap-2">
                  <Textarea
                    value={ex}
                    onChange={(e) =>
                      setExamples((prev) => prev.map((x, i) => (i === idx ? e.target.value : x)))
                    }
                    placeholder={`Example response ${idx + 1}…`}
                    className="flex-1 min-h-[80px]"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => setExamples((prev) => prev.filter((_, i) => i !== idx))}
                    className="self-start mt-1.5 text-muted-foreground hover:text-destructive shrink-0"
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>
              ))}
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setExamples((prev) => [...prev, ''])}
                className="gap-1.5 self-start"
              >
                <Plus className="size-3.5" />
                Add Example
              </Button>
            </div>
          </section>

          {/* ── 12. Default Persona ── */}
          <section className="rounded-xl border border-border bg-card p-5">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-sm font-semibold">Default Persona</h2>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Use this persona when no other is specified.
                </p>
              </div>
              <Switch
                checked={isDefault}
                onCheckedChange={(checked) => setIsDefault(checked)}
              />
            </div>
          </section>

          {/* Mobile: edit-mode actions */}
          {isEditMode && (
            <div className="flex gap-2 sm:hidden">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void handleDuplicate()}
                disabled={isSaving}
                className="gap-1.5"
              >
                <Copy className="size-3.5" />
                Duplicate
              </Button>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => setShowDeleteDialog(true)}
                className="gap-1.5"
              >
                <Trash2 className="size-3.5" />
                Delete
              </Button>
            </div>
          )}
        </div>

        {/* Preview column — desktop only */}
        <div className="hidden md:flex md:flex-col w-[380px] shrink-0 border-l border-border overflow-y-auto p-6">
          {previewPanel}
        </div>
      </div>

      {/* Mobile preview toggle */}
      <div className={cn('md:hidden border-t border-border shrink-0 pb-16')}>
        <button
          type="button"
          onClick={() => setShowMobilePreview((v) => !v)}
          className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium hover:bg-muted/50 transition-colors"
        >
          <span>Live Preview</span>
          {showMobilePreview ? (
            <ChevronUp className="size-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="size-4 text-muted-foreground" />
          )}
        </button>
        {showMobilePreview && (
          <div className="p-4 border-t border-border">{previewPanel}</div>
        )}
      </div>

      {/* Delete confirmation dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={(open) => setShowDeleteDialog(open)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Persona</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{name}"? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleDelete()}
              disabled={isDeleting}
              className="gap-1.5"
            >
              {isDeleting && <Loader2 className="size-3.5 animate-spin" />}
              {isDeleting ? 'Deleting…' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
