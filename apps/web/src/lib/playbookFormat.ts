import type { PlaybookContent } from '../api/playbooks'

function bulletList(items: string[]): string {
  return items.map((i) => `- ${i}`).join('\n')
}

/** Renders a PlaybookContent as markdown — used by "Salin" and "Export .md". */
export function playbookToMarkdown(content: PlaybookContent, version: number): string {
  return [
    `# Playbook v${version}`,
    '',
    '## Ringkasan',
    content.summary,
    '',
    '## Value Proposition',
    content.value_prop,
    '',
    '## Stakeholders',
    bulletList(content.stakeholders),
    '',
    '## Strategi',
    bulletList(content.strategy_checklist),
    '',
    '## Timeline',
    bulletList(content.timeline),
    '',
    '## Risiko',
    bulletList(content.risks),
    '',
    '## Next Actions',
    bulletList(content.next_actions),
    '',
  ].join('\n')
}
