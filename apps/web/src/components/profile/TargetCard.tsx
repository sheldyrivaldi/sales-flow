import { useId } from 'react'
import { Info } from 'lucide-react'
import Card, { CardHeader, CardBody } from '../ui/Card'
import Field from '../ui/Field'
import Input from '../ui/Input'
import Select from '../ui/Select'
import ChipInput from '../ui/ChipInput'
import Tooltip from '../ui/Tooltip'
import {
  COUNTRY_PRESETS,
  INDUSTRY_PRESETS,
  PROCUREMENT_TYPE_PRESETS,
  DOCUMENT_LANGUAGE_PRESETS,
  WORK_MODEL_OPTIONS,
  DECISION_MAKER_PRESETS,
} from '../../lib/profilePresets'
import type { OtakAgentFormState, OtakAgentFormPatch } from './types'

export interface TargetCardProps {
  form: OtakAgentFormState
  onChange: (patch: OtakAgentFormPatch) => void
  disabled?: boolean
  valueMinError?: string
}

export default function TargetCard({ form, onChange, disabled, valueMinError }: TargetCardProps) {
  const titleId = useId()

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-1.5">
          <h2 className="text-body font-semibold text-fg">Target Peluang</h2>
          <Tooltip content="Kriteria yang dipakai AI untuk menyaring tender relevan (negara, industri, nilai, deadline).">
            <Info className="w-3.5 h-3.5 text-fg-subtle" aria-hidden="true" />
          </Tooltip>
        </div>
      </CardHeader>
      <CardBody className="flex flex-col gap-4">
        <Field label="Negara">
          <ChipInput
            value={form.countries}
            onChange={(v) => onChange({ countries: v })}
            presets={COUNTRY_PRESETS}
            disabled={disabled}
            placeholder="Tambah negara…"
          />
        </Field>

        <Field label="Industri">
          <ChipInput
            value={form.industries}
            onChange={(v) => onChange({ industries: v })}
            presets={INDUSTRY_PRESETS}
            disabled={disabled}
            placeholder="Tambah industri…"
          />
        </Field>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Field label="Nilai min" required htmlFor={`${titleId}-value-min`} helper="Rp" error={valueMinError}>
            <Input
              id={`${titleId}-value-min`}
              type="number"
              min="0"
              value={form.valueMin}
              onChange={(e) => onChange({ valueMin: e.target.value })}
              invalid={!!valueMinError}
              disabled={disabled}
            />
          </Field>
          <Field label="Nilai ideal" htmlFor={`${titleId}-value-ideal`} helper="Rp">
            <Input
              id={`${titleId}-value-ideal`}
              type="number"
              min="0"
              value={form.valueIdeal}
              onChange={(e) => onChange({ valueIdeal: e.target.value })}
              disabled={disabled}
            />
          </Field>
          <Field label="Nilai maks" htmlFor={`${titleId}-value-max`} helper="Rp">
            <Input
              id={`${titleId}-value-max`}
              type="number"
              min="0"
              value={form.valueMax}
              onChange={(e) => onChange({ valueMax: e.target.value })}
              disabled={disabled}
            />
          </Field>
          <Field label="Deadline min" htmlFor={`${titleId}-deadline`} helper="hari">
            <Input
              id={`${titleId}-deadline`}
              type="number"
              min="0"
              value={form.deadlineMinDays}
              onChange={(e) => onChange({ deadlineMinDays: e.target.value })}
              disabled={disabled}
            />
          </Field>
        </div>

        <Field label="Jenis pengadaan">
          <ChipInput
            value={form.procurementTypes}
            onChange={(v) => onChange({ procurementTypes: v })}
            presets={PROCUREMENT_TYPE_PRESETS}
            disabled={disabled}
            placeholder="Tambah jenis…"
          />
        </Field>

        <Field label="Ukuran buyer" htmlFor={`${titleId}-buyer-size`} helper="Revenue min, jumlah karyawan, atau skala operasi">
          <Input
            id={`${titleId}-buyer-size`}
            value={form.buyerSizeNote}
            onChange={(e) => onChange({ buyerSizeNote: e.target.value })}
            disabled={disabled}
            placeholder="mis. Revenue ≥ Rp 30 miliar"
          />
        </Field>

        <Field label="Bahasa dokumen tender">
          <ChipInput
            value={form.documentLanguages}
            onChange={(v) => onChange({ documentLanguages: v })}
            presets={DOCUMENT_LANGUAGE_PRESETS}
            disabled={disabled}
            placeholder="Tambah bahasa…"
          />
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Model kerja" htmlFor={`${titleId}-work-model`}>
            <Select
              id={`${titleId}-work-model`}
              value={form.workModel}
              onChange={(e) => onChange({ workModel: e.target.value })}
              disabled={disabled}
            >
              <option value="">— Pilih —</option>
              {WORK_MODEL_OPTIONS.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </Select>
          </Field>
          {form.workModel && form.workModel !== 'Remote' && (
            <Field label="Batasan onsite" htmlFor={`${titleId}-onsite-limit`}>
              <Input
                id={`${titleId}-onsite-limit`}
                value={form.onsiteLimitNote}
                onChange={(e) => onChange({ onsiteLimitNote: e.target.value })}
                disabled={disabled}
                placeholder="mis. maks 2 hari/minggu, Jabodetabek"
              />
            </Field>
          )}
        </div>

        <Field label="Target decision maker" helper="Peran yang dituju untuk contact enrichment">
          <ChipInput
            value={form.decisionMakerRoles}
            onChange={(v) => onChange({ decisionMakerRoles: v })}
            presets={DECISION_MAKER_PRESETS}
            disabled={disabled}
            placeholder="Tambah peran…"
          />
        </Field>
      </CardBody>
    </Card>
  )
}
