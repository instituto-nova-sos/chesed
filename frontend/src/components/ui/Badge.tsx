const roleColors: Record<string, string> = {
  VOLUNTEER: 'bg-green-100 text-green-800',
  ASSISTED: 'bg-blue-100 text-blue-800',
  PROFESSIONAL: 'bg-purple-100 text-purple-800',
  COORDINATOR: 'bg-orange-100 text-orange-800',
  ADMIN: 'bg-red-100 text-red-800',
};

interface BadgeProps {
  label: string;
  variant?: string;
}

export function Badge({ label, variant }: BadgeProps) {
  const colors = (variant && roleColors[variant]) || 'bg-gray-100 text-gray-800';
  return (
    <span
      className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${colors}`}
    >
      {label}
    </span>
  );
}
