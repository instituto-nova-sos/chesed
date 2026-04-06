import { useAuth } from '../../hooks/useAuth';

function MenuIcon() {
  return (
    <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M4 6h16M4 12h16M4 18h16"
      />
    </svg>
  );
}

interface HeaderProps {
  onMenuToggle: () => void;
}

export function Header({ onMenuToggle }: HeaderProps) {
  const { email, roles, logout } = useAuth();

  const primaryRole = roles[0] ?? 'User';

  return (
    <header className="flex h-16 items-center justify-between border-b bg-white px-4 shadow-sm">
      <button
        type="button"
        onClick={onMenuToggle}
        className="rounded-lg p-2 text-gray-500 hover:bg-gray-100 md:hidden"
        aria-label="Toggle menu"
      >
        <MenuIcon />
      </button>

      <div className="hidden md:block" />

      <div className="flex items-center gap-4">
        <div className="text-right">
          <p className="text-sm font-medium text-gray-700">{email}</p>
          <span className="inline-block rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800">
            {primaryRole}
          </span>
        </div>

        <button
          type="button"
          onClick={logout}
          className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50"
        >
          Sair
        </button>
      </div>
    </header>
  );
}
