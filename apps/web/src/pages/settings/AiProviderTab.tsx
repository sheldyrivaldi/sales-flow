import { useState } from 'react'
import { Sparkles } from 'lucide-react'
import Card, { CardBody } from '../../components/ui/Card'
import Badge from '../../components/ui/Badge'
import Button from '../../components/ui/Button'
import Field from '../../components/ui/Field'
import Select from '../../components/ui/Select'
import Input from '../../components/ui/Input'
import Skeleton from '../../components/ui/Skeleton'
import { toast } from '../../lib/toast'
import { useAISetting, useUpdateAISetting, useTestAISetting } from '../../api/settings'
import type { AIProvider } from '../../api/settings'

const MODEL_PRESETS: Record<AIProvider, string[]> = {
  openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4.1', 'o3-mini'],
  openrouter: [
    'anthropic/claude-sonnet-4-6',
    'anthropic/claude-opus-4-8',
    'openai/gpt-4o',
    'google/gemini-2.5-pro',
  ],
}

const CUSTOM_SENTINEL = '__custom__'

export default function AiProviderTab() {
  const { data: setting, isLoading } = useAISetting()
  const update = useUpdateAISetting()
  const test = useTestAISetting()

  const [provider, setProvider] = useState<AIProvider>('openai')
  const [modelChoice, setModelChoice] = useState<string>(MODEL_PRESETS.openai[0])
  const [customModel, setCustomModel] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [toolsetsInput, setToolsetsInput] = useState('')

  // isDirty tracks whether the admin has started editing since the form was
  // last synced from the server. A background refetch (staleTime elapsed,
  // window refocus) hands back a new `setting` object even when the data is
  // unchanged — without this guard, the sync block below would fire again
  // and silently discard in-progress unsaved edits.
  const [isDirty, setIsDirty] = useState(false)

  // Sync form state from the fetched config once it arrives — done during
  // render (React's documented pattern for "adjusting state when a prop/
  // query changes") rather than in a useEffect, which would cause an extra
  // render pass. syncedSetting tracks which object identity we've already
  // applied, so this only fires once per fetch, not on every render — and is
  // skipped entirely while the admin has unsaved edits (isDirty).
  const [syncedSetting, setSyncedSetting] = useState(setting)
  if (setting !== syncedSetting && !isDirty) {
    setSyncedSetting(setting)
    if (setting && setting.provider) {
      setProvider(setting.provider)
      const presets = MODEL_PRESETS[setting.provider]
      if (presets.includes(setting.model)) {
        setModelChoice(setting.model)
        setCustomModel('')
      } else {
        setModelChoice(CUSTOM_SENTINEL)
        setCustomModel(setting.model)
      }
      setBaseUrl(setting.base_url ?? '')
      setToolsetsInput((setting.enabled_toolsets ?? []).join(', '))
    }
  }

  const effectiveModel = modelChoice === CUSTOM_SENTINEL ? customModel : modelChoice

  function edit<T>(setter: (v: T) => void) {
    return (v: T) => {
      setIsDirty(true)
      setter(v)
    }
  }
  const onProviderChange = edit(setProvider)
  const onModelChoiceChange = edit(setModelChoice)
  const onCustomModelChange = edit(setCustomModel)
  const onBaseUrlChange = edit(setBaseUrl)
  const onApiKeyChange = edit(setApiKey)
  const onToolsetsInputChange = edit(setToolsetsInput)

  async function handleSave() {
    if (!effectiveModel.trim()) {
      toast.error('Model wajib diisi.')
      return
    }
    try {
      await update.mutateAsync({
        provider,
        model: effectiveModel.trim(),
        base_url: baseUrl.trim() || undefined,
        api_key: apiKey.trim() || undefined,
        enabled_toolsets: toolsetsInput
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
      })
      toast.success('Konfigurasi AI Provider tersimpan.')
      setApiKey('')
      // Saved successfully — allow the next refetch (triggered by this
      // mutation's own invalidateQueries) to sync the form again, e.g. to
      // pick up the freshly recomputed api_key_masked.
      setIsDirty(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Gagal menyimpan konfigurasi, coba lagi nanti.')
    }
  }

  async function handleTest() {
    try {
      const res = await test.mutateAsync()
      if (res.status === 'ok') {
        toast.success(`Test koneksi berhasil${res.version ? ` • v${res.version}` : ''}.`)
      } else {
        toast.error('Test koneksi gagal. Periksa provider/model/API key.')
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Test koneksi gagal, coba lagi nanti.')
    }
  }

  if (isLoading) {
    return (
      <Card className="max-w-lg">
        <CardBody className="flex flex-col gap-3">
          <Skeleton variant="text" className="h-5 w-1/2" />
          <Skeleton variant="text" className="h-10 w-full" />
          <Skeleton variant="text" className="h-10 w-full" />
        </CardBody>
      </Card>
    )
  }

  return (
    <Card className="max-w-lg">
      <CardBody className="flex flex-col gap-4">
        {setting?.is_active && (
          <div>
            <Badge tone="success">
              Aktif • {setting.provider} • {setting.model}
            </Badge>
          </div>
        )}

        <Field label="Provider" htmlFor="ai-provider" required>
          <Select
            id="ai-provider"
            value={provider}
            onChange={(e) => {
              const p = e.target.value as AIProvider
              onProviderChange(p)
              onModelChoiceChange(MODEL_PRESETS[p][0])
            }}
          >
            <option value="openai">OpenAI</option>
            <option value="openrouter">OpenRouter</option>
          </Select>
        </Field>

        <Field label="Model" htmlFor="ai-model" required>
          <Select id="ai-model" value={modelChoice} onChange={(e) => onModelChoiceChange(e.target.value)}>
            {MODEL_PRESETS[provider].map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
            <option value={CUSTOM_SENTINEL}>Custom…</option>
          </Select>
        </Field>

        {modelChoice === CUSTOM_SENTINEL && (
          <Field label="Nama model custom" htmlFor="ai-model-custom">
            <Input
              id="ai-model-custom"
              value={customModel}
              onChange={(e) => onCustomModelChange(e.target.value)}
              placeholder="mis. anthropic/claude-opus-4-8"
            />
          </Field>
        )}

        <Field
          label="Base URL (opsional)"
          htmlFor="ai-base-url"
          helper="Kosongkan untuk memakai default provider."
        >
          <Input
            id="ai-base-url"
            value={baseUrl}
            onChange={(e) => onBaseUrlChange(e.target.value)}
            placeholder="https://api.openai.com/v1"
          />
        </Field>

        <Field
          label="API Key"
          htmlFor="ai-api-key"
          helper={
            setting?.api_key_masked
              ? `Tersimpan: ${setting.api_key_masked}. Kosongkan untuk tetap memakai key ini.`
              : 'Wajib diisi untuk konfigurasi pertama.'
          }
        >
          <Input
            id="ai-api-key"
            type="password"
            value={apiKey}
            onChange={(e) => onApiKeyChange(e.target.value)}
            placeholder={setting?.api_key_masked || 'sk-...'}
            autoComplete="off"
          />
        </Field>

        <Field
          label="Enabled toolsets (opsional)"
          htmlFor="ai-toolsets"
          helper="Pisahkan dengan koma, mis. web, search"
        >
          <Input
            id="ai-toolsets"
            value={toolsetsInput}
            onChange={(e) => onToolsetsInputChange(e.target.value)}
            placeholder="web, search"
          />
        </Field>

        <div className="flex gap-2">
          <Button loading={update.isPending} onClick={handleSave}>
            Simpan
          </Button>
          <Button variant="secondary" loading={test.isPending} onClick={handleTest}>
            Test Koneksi
          </Button>
        </div>

        <p className="flex items-center gap-1 text-caption text-fg-muted">
          <Sparkles className="w-3.5 h-3.5 text-accent" aria-hidden="true" />
          Perubahan di sini langsung berlaku untuk chat/scoring berikutnya, tanpa restart.
        </p>
      </CardBody>
    </Card>
  )
}
