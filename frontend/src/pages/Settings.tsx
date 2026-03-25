import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import type { RecordModel } from 'pocketbase'
import { useNavigate } from 'react-router'
import pb from '@/lib/pocketbase'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Slider } from '@/components/ui/slider'
import { CheckCircle2, XCircle, ExternalLink, Plus, Pencil, Trash2 } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface UserSettings extends RecordModel {
  discord_webhook_url: string
  min_relevance_score: number
  monitoring_interval_minutes: number
}

interface RedditStatus {
  connected: boolean
  username?: string
}

interface PersonaTraits {
  formality?: number
  humor?: number
  verbosity?: number
  empathy?: number
  confidence?: number
}

interface PersonaItem extends RecordModel {
  name: string
  description: string
  traits: PersonaTraits
  is_default: boolean
}

interface AdminUser {
  id: string
  email: string
  created: string
}

export default function Settings() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [redditStatus, setRedditStatus] = useState<RedditStatus>({ connected: false })
  const [userSettings, setUserSettings] = useState<UserSettings | null>(null)
  const [webhookUrl, setWebhookUrl] = useState('')
  const [minScore, setMinScore] = useState(50)
  const [monitoringInterval, setMonitoringInterval] = useState(15)
  const [isSavingWebhook, setIsSavingWebhook] = useState(false)
  const [isSavingMonitoring, setIsSavingMonitoring] = useState(false)
  const [isDisconnecting, setIsDisconnecting] = useState(false)
  const [webhookSaved, setWebhookSaved] = useState(false)
  const [personas, setPersonas] = useState<PersonaItem[]>([])

  const [isAdmin, setIsAdmin] = useState(false)
  const [adminUsers, setAdminUsers] = useState<AdminUser[]>([])
  const [newUserEmail, setNewUserEmail] = useState('')
  const [newUserPassword, setNewUserPassword] = useState('')
  const [isCreatingUser, setIsCreatingUser] = useState(false)
  const [userMgmtError, setUserMgmtError] = useState('')
  const [userMgmtSuccess, setUserMgmtSuccess] = useState('')

  const [productName, setProductName] = useState('')
  const [productDescription, setProductDescription] = useState('')
  const [productTargetAudience, setProductTargetAudience] = useState('')
  const [productKeyFeatures, setProductKeyFeatures] = useState('')
  const [productDifferentiators, setProductDifferentiators] = useState('')
  const [productWebsiteUrl, setProductWebsiteUrl] = useState('')
  const [isSavingProduct, setIsSavingProduct] = useState(false)
  const [productSaved, setProductSaved] = useState(false)

  const [availableModels, setAvailableModels] = useState<string[]>([])
  const [aiModel, setAiModel] = useState('')
  const [customModelName, setCustomModelName] = useState('')
  const [isSavingAiModel, setIsSavingAiModel] = useState(false)
  const [aiModelSaved, setAiModelSaved] = useState(false)

  const fetchAdminUsers = async () => {
    try {
      const res = await fetch('/api/admin/users', {
        headers: { Authorization: pb.authStore.token },
      })
      if (res.status === 403) {
        setIsAdmin(false)
        return
      }
      const users = await res.json() as AdminUser[]
      setIsAdmin(true)
      setAdminUsers(users)
    } catch (_e) {
      void _e
    }
  }

  useEffect(() => {
    const fetchAll = async () => {
      try {
        const res = await fetch('/api/reddit/status', {
          headers: { Authorization: pb.authStore.token },
        })
        const data = (await res.json()) as RedditStatus
        setRedditStatus(data)
      } catch (_e) {
        void _e
      }

      try {
        const settings = await pb.collection('user_settings').getFirstListItem<UserSettings>(
          `user = "${user?.id ?? ''}"`
        )
        setUserSettings(settings)
        setWebhookUrl(settings.discord_webhook_url ?? '')
        setMinScore(settings.min_relevance_score ?? 50)
        setMonitoringInterval(settings.monitoring_interval_minutes ?? 15)
      } catch (_e) {
        void _e
      }

      try {
        const list = await pb.collection('personas').getFullList<PersonaItem>({
          filter: `created_by = "${user?.id ?? ''}"`,
          sort: '-created',
        })
        setPersonas(list)
      } catch (_e) {
        void _e
      }

      await fetchAdminUsers()

      try {
        const productRecord = await pb.collection('product_context').getFirstListItem('id != ""')
        setProductName(productRecord.name ?? '')
        setProductDescription(productRecord.description ?? '')
        setProductTargetAudience(productRecord.target_audience ?? '')
        setProductKeyFeatures(productRecord.key_features ?? '')
        setProductDifferentiators(productRecord.differentiators ?? '')
        setProductWebsiteUrl(productRecord.website_url ?? '')
      } catch (_e) {
        void _e
      }

      let models: string[] = []
      try {
        const res = await fetch('/api/ai/models', {
          headers: { Authorization: pb.authStore.token },
        })
        if (res.ok) {
          models = (await res.json()) as string[]
          setAvailableModels(models)
        }
      } catch (_e) {
        void _e
      }

      try {
        const record = await pb.collection('settings').getFirstListItem('key = "ai_model"')
        const val = record.value as string
        const parsed = JSON.parse(val) as string
        if (models.length === 0 || models.includes(parsed)) {
          setAiModel(parsed)
        } else {
          setAiModel('custom')
          setCustomModelName(parsed)
        }
      } catch (_e) {
        void _e
      }
    }
    void fetchAll()
  }, [user?.id])

  const handleSaveProduct = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsSavingProduct(true)
    try {
      const data = {
        name: productName,
        description: productDescription,
        target_audience: productTargetAudience,
        key_features: productKeyFeatures,
        differentiators: productDifferentiators,
        website_url: productWebsiteUrl,
      }
      try {
        const existing = await pb.collection('product_context').getFirstListItem('id != ""')
        await pb.collection('product_context').update(existing.id, data)
      } catch {
        await pb.collection('product_context').create(data)
      }
      setProductSaved(true)
      setTimeout(() => setProductSaved(false), 2000)
    } catch (_e) {
      void _e
    } finally {
      setIsSavingProduct(false)
    }
  }

  const handleSaveAiModel = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsSavingAiModel(true)
    const modelName = aiModel === 'custom' ? customModelName : aiModel
    try {
      try {
        const existing = await pb.collection('settings').getFirstListItem('key = "ai_model"')
        await pb.collection('settings').update(existing.id, { value: JSON.stringify(modelName) })
      } catch {
        await pb.collection('settings').create({ key: 'ai_model', value: JSON.stringify(modelName) })
      }
      setAiModelSaved(true)
      setTimeout(() => setAiModelSaved(false), 2000)
    } catch (_e) {
      void _e
    } finally {
      setIsSavingAiModel(false)
    }
  }

  const handleConnectReddit = () => {
    window.location.href = '/api/reddit/auth'
  }

  const handleDisconnectReddit = async () => {
    setIsDisconnecting(true)
    try {
      await fetch('/api/reddit/disconnect', {
        method: 'POST',
        headers: { Authorization: pb.authStore.token },
      })
      setRedditStatus({ connected: false })
    } catch (_e) {
      void _e
    } finally {
      setIsDisconnecting(false)
    }
  }

  const handleSaveWebhook = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsSavingWebhook(true)
    try {
      if (userSettings) {
        const updated = await pb.collection('user_settings').update<UserSettings>(userSettings.id, {
          discord_webhook_url: webhookUrl,
        })
        setUserSettings(updated)
      } else {
        const created = await pb.collection('user_settings').create<UserSettings>({
          discord_webhook_url: webhookUrl,
          user: user?.id,
        })
        setUserSettings(created)
      }
      setWebhookSaved(true)
      setTimeout(() => setWebhookSaved(false), 2000)
    } catch (_e) {
      void _e
    } finally {
      setIsSavingWebhook(false)
    }
  }

  const handleSaveMonitoring = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setIsSavingMonitoring(true)
    try {
      if (userSettings) {
        const updated = await pb.collection('user_settings').update<UserSettings>(userSettings.id, {
          min_relevance_score: minScore,
          monitoring_interval_minutes: monitoringInterval,
        })
        setUserSettings(updated)
      } else {
        const created = await pb.collection('user_settings').create<UserSettings>({
          min_relevance_score: minScore,
          monitoring_interval_minutes: monitoringInterval,
          user: user?.id,
        })
        setUserSettings(created)
      }
    } catch (_e) {
      void _e
    } finally {
      setIsSavingMonitoring(false)
    }
  }

  const handleCreateUser = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setUserMgmtError('')
    setUserMgmtSuccess('')
    setIsCreatingUser(true)
    try {
      const res = await fetch('/api/admin/users', {
        method: 'POST',
        headers: {
          Authorization: pb.authStore.token,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email: newUserEmail, password: newUserPassword }),
      })
      if (!res.ok) {
        const data = await res.json() as { error?: string }
        setUserMgmtError(data.error ?? 'Failed to create user.')
        return
      }
      setNewUserEmail('')
      setNewUserPassword('')
      setUserMgmtSuccess('User created successfully.')
      setTimeout(() => setUserMgmtSuccess(''), 3000)
      await fetchAdminUsers()
    } catch (_e) {
      void _e
      setUserMgmtError('Failed to create user. Please try again.')
    } finally {
      setIsCreatingUser(false)
    }
  }

  const handleDeleteUser = async (userId: string) => {
    if (!window.confirm('Are you sure you want to delete this user?')) return
    try {
      await fetch(`/api/admin/users/${userId}`, {
        method: 'DELETE',
        headers: { Authorization: pb.authStore.token },
      })
      await fetchAdminUsers()
    } catch (_e) {
      void _e
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur border-b border-border px-4 py-4">
        <h1 className="text-xl font-bold">Settings</h1>
      </div>

      <div className="flex-1 overflow-auto p-4 flex flex-col gap-6 max-w-lg">
        <section className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-semibold mb-1">Reddit Account</h2>
          <p className="text-xs text-muted-foreground mb-4">
            Connect your Reddit account to enable posting replies.
          </p>

          <div className="flex items-center gap-3 mb-4">
            {redditStatus.connected ? (
              <>
                <CheckCircle2 className="size-4 text-green-500 shrink-0" />
                <div>
                  <Badge variant="secondary" className="text-xs">Connected</Badge>
                  {redditStatus.username && (
                    <span className="ml-2 text-sm text-muted-foreground">
                      u/{redditStatus.username}
                    </span>
                  )}
                </div>
              </>
            ) : (
              <>
                <XCircle className="size-4 text-muted-foreground/50 shrink-0" />
                <Badge variant="outline" className="text-xs">Disconnected</Badge>
              </>
            )}
          </div>

          {redditStatus.connected ? (
            <Button
              variant="destructive"
              size="sm"
              onClick={() => void handleDisconnectReddit()}
              disabled={isDisconnecting}
            >
              {isDisconnecting ? 'Disconnecting…' : 'Disconnect Reddit'}
            </Button>
          ) : (
            <Button size="sm" onClick={handleConnectReddit} className="gap-2">
              <ExternalLink className="size-3.5" />
              Connect Reddit
            </Button>
          )}
        </section>

        <Separator />

        <section className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-semibold mb-1">My Personas</h2>
          <p className="text-xs text-muted-foreground mb-4">
            Manage your response personas for different goals and audiences.
          </p>

          {personas.length > 0 && (
            <div className="flex flex-col gap-2 mb-3">
              {personas.map((p) => (
                <div
                  key={p.id}
                  className="flex items-start justify-between gap-3 rounded-lg border border-border bg-muted/20 px-3 py-3"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className="text-sm font-medium truncate">{p.name}</p>
                      {p.is_default && (
                        <Badge variant="default" className="text-xs shrink-0">Default</Badge>
                      )}
                    </div>
                    {p.description && (
                      <p className="text-xs text-muted-foreground mt-0.5 truncate">{p.description}</p>
                    )}
                    {p.traits && (
                      <div className="flex flex-wrap gap-1 mt-1.5">
                        {(Object.entries(p.traits) as Array<[string, number | undefined]>)
                          .filter(([, v]) => v !== undefined)
                          .slice(0, 3)
                          .map(([k, v]) => (
                            <Badge key={k} variant="outline" className="text-xs px-1.5 py-0">
                              {k}: {v}
                            </Badge>
                          ))}
                      </div>
                    )}
                  </div>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => navigate(`/personas/${p.id}`)}
                    className="shrink-0 text-muted-foreground hover:text-foreground mt-0.5"
                  >
                    <Pencil className="size-3.5" />
                  </Button>
                </div>
              ))}
            </div>
          )}

          <Button
            size="sm"
            variant="outline"
            onClick={() => navigate('/personas/new')}
            className="gap-1.5"
          >
            <Plus className="size-3.5" />
            New Persona
          </Button>
        </section>

        <Separator />

        <section className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-semibold mb-1">Discord Webhook</h2>
          <p className="text-xs text-muted-foreground mb-4">
            Receive notifications in Discord when new relevant threads are found.
          </p>
          <form onSubmit={(e) => void handleSaveWebhook(e)} className="flex flex-col gap-3">
            <Input
              type="url"
              placeholder="https://discord.com/api/webhooks/…"
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
            />
            <Button type="submit" size="sm" disabled={isSavingWebhook} className="self-start">
              {webhookSaved ? 'Saved!' : isSavingWebhook ? 'Saving…' : 'Save Webhook'}
            </Button>
          </form>
        </section>

        <Separator />

        <section className="rounded-xl border border-border bg-card p-5">
          <h2 className="text-sm font-semibold mb-1">Monitoring</h2>
          <p className="text-xs text-muted-foreground mb-4">
            Configure relevance thresholds and scan frequency.
          </p>
          <form onSubmit={(e) => void handleSaveMonitoring(e)} className="flex flex-col gap-5">
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium">Min Relevance Score</label>
                <span className="text-sm font-mono text-primary">{minScore}%</span>
              </div>
              <Slider
                min={0}
                max={100}
                step={5}
                value={[minScore]}
                onValueChange={(val) => {
                  const v = Array.isArray(val) ? val[0] : val
                  if (typeof v === 'number') setMinScore(v)
                }}
              />
              <p className="text-xs text-muted-foreground">
                Threads below this score will be filtered from the inbox.
              </p>
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-medium">Monitoring Interval (minutes)</label>
              <Input
                type="number"
                min={1}
                max={1440}
                value={monitoringInterval}
                onChange={(e) => setMonitoringInterval(Number(e.target.value))}
                className="w-32"
              />
            </div>

            <Button type="submit" size="sm" disabled={isSavingMonitoring} className="self-start">
              {isSavingMonitoring ? 'Saving…' : 'Save Settings'}
            </Button>
          </form>
        </section>

        {isAdmin && (
          <>
            <Separator />

            <section className="rounded-xl border border-border bg-card p-5">
              <h2 className="text-sm font-semibold mb-1">Product</h2>
              <p className="text-xs text-muted-foreground mb-4">
                Describe your product so AI can score threads and generate replies with context.
              </p>
              <form onSubmit={(e) => void handleSaveProduct(e)} className="flex flex-col gap-3">
                <Input
                  type="text"
                  placeholder="Product name"
                  value={productName}
                  onChange={(e) => setProductName(e.target.value)}
                />
                <Textarea
                  placeholder="What does your product do?"
                  value={productDescription}
                  onChange={(e) => setProductDescription(e.target.value)}
                />
                <Textarea
                  placeholder="Who is your ideal customer?"
                  value={productTargetAudience}
                  onChange={(e) => setProductTargetAudience(e.target.value)}
                />
                <Textarea
                  placeholder="Main features and capabilities"
                  value={productKeyFeatures}
                  onChange={(e) => setProductKeyFeatures(e.target.value)}
                />
                <Textarea
                  placeholder="What makes you different from competitors?"
                  value={productDifferentiators}
                  onChange={(e) => setProductDifferentiators(e.target.value)}
                />
                <Input
                  type="url"
                  placeholder="https://..."
                  value={productWebsiteUrl}
                  onChange={(e) => setProductWebsiteUrl(e.target.value)}
                />
                <Button type="submit" size="sm" disabled={isSavingProduct} className="self-start">
                  {productSaved ? 'Saved!' : isSavingProduct ? 'Saving…' : 'Save Product'}
                </Button>
              </form>
            </section>

            <Separator />

            <section className="rounded-xl border border-border bg-card p-5">
              <h2 className="text-sm font-semibold mb-1">AI Model</h2>
              <p className="text-xs text-muted-foreground mb-4">
                Select the AI model used for scoring and response generation.
              </p>
              <form onSubmit={(e) => void handleSaveAiModel(e)} className="flex flex-col gap-3">
                <Select value={aiModel} onValueChange={(v) => { if (v !== null) setAiModel(v) }}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select model…" />
                  </SelectTrigger>
                  <SelectContent>
                    {availableModels.map((m) => (
                      <SelectItem key={m} value={m}>{m}</SelectItem>
                    ))}
                    <SelectItem value="custom">Custom…</SelectItem>
                  </SelectContent>
                </Select>
                {aiModel === 'custom' && (
                  <Input
                    placeholder="Enter model name…"
                    value={customModelName}
                    onChange={(e) => setCustomModelName(e.target.value)}
                    required
                  />
                )}
                <Button type="submit" size="sm" disabled={isSavingAiModel} className="self-start">
                  {aiModelSaved ? 'Saved!' : isSavingAiModel ? 'Saving…' : 'Save Model'}
                </Button>
              </form>
            </section>

            <Separator />

            <section className="rounded-xl border border-border bg-card p-5">
              <h2 className="text-sm font-semibold mb-1">User Management</h2>
              <p className="text-xs text-muted-foreground mb-4">
                Create and manage user accounts.
              </p>

              {adminUsers.length > 0 && (
                <div className="flex flex-col gap-2 mb-5">
                  {adminUsers.map((u) => (
                    <div
                      key={u.id}
                      className="flex items-center justify-between gap-3 rounded-lg border border-border bg-muted/20 px-3 py-2.5"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{u.email}</p>
                        <p className="text-xs text-muted-foreground">
                          {new Date(u.created).toLocaleDateString()}
                        </p>
                      </div>
                      {u.id !== user?.id && (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => void handleDeleteUser(u.id)}
                          className="shrink-0 text-muted-foreground hover:text-destructive"
                        >
                          <Trash2 className="size-3.5" />
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              )}

              <form onSubmit={(e) => void handleCreateUser(e)} className="flex flex-col gap-3">
                <Input
                  type="email"
                  placeholder="user@example.com"
                  required
                  value={newUserEmail}
                  onChange={(e) => setNewUserEmail(e.target.value)}
                />
                <Input
                  type="password"
                  placeholder="Password"
                  required
                  value={newUserPassword}
                  onChange={(e) => setNewUserPassword(e.target.value)}
                />

                {userMgmtError && (
                  <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                    {userMgmtError}
                  </p>
                )}
                {userMgmtSuccess && (
                  <p className="rounded-md bg-green-500/10 px-3 py-2 text-sm text-green-600 dark:text-green-400">
                    {userMgmtSuccess}
                  </p>
                )}

                <Button type="submit" size="sm" disabled={isCreatingUser} className="self-start gap-1.5">
                  <Plus className="size-3.5" />
                  {isCreatingUser ? 'Creating…' : 'Create User'}
                </Button>
              </form>
            </section>
          </>
        )}
      </div>
    </div>
  )
}
