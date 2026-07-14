package dto

type PipelineStageResponse struct {
	Stage      string  `json:"stage"`
	Count      int64   `json:"count"`
	TotalValue float64 `json:"total_value"`
}

type DashboardSummaryResponse struct {
	Pipeline            []PipelineStageResponse `json:"pipeline"`
	TotalPipelineCount  int64                   `json:"total_pipeline_count"`
	TotalPipelineValue  float64                 `json:"total_pipeline_value"`
	PriorityTenders     []TenderResponse        `json:"priority_tenders"`
	DiscoveryTodayCount int64                   `json:"discovery_today_count"`
}
