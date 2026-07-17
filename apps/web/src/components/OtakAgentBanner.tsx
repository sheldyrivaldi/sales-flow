import { Sparkles } from 'lucide-react'
import { Link } from 'react-router'
import { useProfile, isProfileConfigured } from '../api/profile'

/**
 * Shown wherever the user might land after skipping onboarding (Dashboard).
 * Renders nothing once a profile version has been saved (version > 0).
 */
export default function OtakAgentBanner() {
  const { data: profile } = useProfile()

  if (isProfileConfigured(profile)) return null

  return (
    <div className="flex items-center justify-between gap-4 rounded-card border border-accent/30 bg-accent/5 px-4 py-3">
      <div className="flex items-center gap-2">
        <Sparkles className="w-5 h-5 text-accent shrink-0" aria-hidden="true" />
        <p className="text-body text-fg">
          Lengkapi Profil Perusahaan agar AI bisa mencari tender untukmu.
        </p>
      </div>
      <Link
        to="/onboarding"
        className="text-body font-medium text-primary hover:underline whitespace-nowrap"
      >
        Lengkapi sekarang
      </Link>
    </div>
  )
}
