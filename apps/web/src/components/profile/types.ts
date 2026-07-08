// Local (string-based, form-friendly) mirror of api/profile.ts Profile — used
// as the single controlled form state shared across all 6 Otak Agent cards.
export interface OtakAgentFormState {
  companyName: string
  oneLiner: string
  serviceCategories: string[]
  techStack: string[]
  countries: string[]
  industries: string[]
  valueMin: string
  valueIdeal: string
  deadlineMinDays: string
  procurementTypes: string[]
  presetFlags: string[]
  customNoGo: string[]
  keywords: string[]
  negativeKeywords: string[]
}

export type OtakAgentFormPatch = Partial<OtakAgentFormState>
