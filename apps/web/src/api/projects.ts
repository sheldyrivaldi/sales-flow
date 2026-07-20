import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiFetch, buildQueryString } from '../lib/api'

// ── Types ─────────────────────────────────────────────────────────────────────

export type ProjectStatus = 'ON_TRACK' | 'AT_RISK' | 'DELAYED' | 'COMPLETED'

export const PROJECT_STATUS_LABELS: Record<ProjectStatus, string> = {
  ON_TRACK: 'On Track',
  AT_RISK: 'Berisiko',
  DELAYED: 'Terlambat',
  COMPLETED: 'Selesai',
}

export interface ProjectMilestone {
  title: string
  due_date?: string | null
  done: boolean
}

export interface ProjectActivity {
  date: string
  note: string
}

export interface Project {
  id: string
  name: string
  client_name: string | null
  contract_value: number | null
  currency: string
  start_date: string | null
  end_date: string | null
  status: ProjectStatus
  progress: number
  description: string | null
  milestones: ProjectMilestone[]
  activities: ProjectActivity[]
  source_type: string | null
  source_id: string | null
  created_at: string
  updated_at: string
}

export interface ProjectUpsertBody {
  name: string
  client_name?: string | null
  contract_value?: number | null
  currency?: string
  start_date?: string | null
  end_date?: string | null
  status?: ProjectStatus
  progress?: number
  description?: string | null
  milestones?: ProjectMilestone[]
}

export interface ProjectSummary {
  total_active: number
  total_value: number
  on_track: number
  at_risk: number
  delayed: number
  completed: number
  avg_progress: number
  ending_soon: number
}

interface ProjectListResponse {
  items: Project[]
  total: number
  page: number
  page_size: number
}

// ── Hooks ─────────────────────────────────────────────────────────────────────

export function useProjects(filters: { status?: ProjectStatus; search?: string } = {}) {
  return useQuery({
    queryKey: ['projects', filters],
    queryFn: () =>
      apiFetch<ProjectListResponse>(`/api/projects${buildQueryString({ ...filters, page_size: 200 })}`),
  })
}

export function useProjectSummary() {
  return useQuery({
    queryKey: ['projects-summary'],
    queryFn: () => apiFetch<ProjectSummary>('/api/projects/summary'),
  })
}

function useInvalidateProjects() {
  const qc = useQueryClient()
  return () => {
    void qc.invalidateQueries({ queryKey: ['projects'] })
    void qc.invalidateQueries({ queryKey: ['projects-summary'] })
  }
}

export function useCreateProject() {
  const invalidate = useInvalidateProjects()
  return useMutation({
    mutationFn: (body: ProjectUpsertBody) =>
      apiFetch<Project>('/api/projects', { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: invalidate,
  })
}

export function useUpdateProject() {
  const invalidate = useInvalidateProjects()
  return useMutation({
    mutationFn: ({ id, ...body }: ProjectUpsertBody & { id: string }) =>
      apiFetch<Project>(`/api/projects/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    onSuccess: invalidate,
  })
}

export function useDeleteProject() {
  const invalidate = useInvalidateProjects()
  return useMutation({
    mutationFn: (id: string) => apiFetch<void>(`/api/projects/${id}`, { method: 'DELETE' }),
    onSuccess: invalidate,
  })
}

export function useAddProjectActivity() {
  const invalidate = useInvalidateProjects()
  return useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) =>
      apiFetch<Project>(`/api/projects/${id}/activities`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      }),
    onSuccess: invalidate,
  })
}
