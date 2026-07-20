// Local (string-based, form-friendly) mirror of api/profile.ts Profile — used
// as the single controlled form state shared across all Otak Agent cards.
export interface OtakAgentFormState {
  companyName: string
  oneLiner: string
  serviceCategories: string[]
  techStack: string[]
  products: string[]
  portfolioRefs: string[]
  supportDocuments: string[]
  vision: string
  mission: string
  countries: string[]
  industries: string[]
  valueMin: string
  valueIdeal: string
  valueMax: string
  deadlineMinDays: string
  procurementTypes: string[]
  buyerSizeNote: string
  documentLanguages: string[]
  workModel: string
  onsiteLimitNote: string
  decisionMakerRoles: string[]
  presetFlags: string[]
  customNoGo: string[]
  keywords: string[]
  negativeKeywords: string[]
  crawlEnabled: boolean
  crawlFrequency: string
  sourceDocRefs: string[]
  // Scoring (RFI §8) — weights as strings for controlled number inputs,
  // same convention as valueMin/valueIdeal above.
  weightCapabilityFit: string
  weightPortfolioMatch: string
  weightCommercialAttractiveness: string
  weightEligibilityFit: string
  weightDeadlineFeasibility: string
  weightStrategicAccountValue: string
  weightDeliveryRisk: string
  weightCompetitionWinProbability: string
  thresholdPursue: string
  thresholdReview: string
  thresholdWatchlist: string
}

export type OtakAgentFormPatch = Partial<OtakAgentFormState>
