import React, { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Logo } from '@/components/Logo';
import { useAuth } from '@/components/AuthProvider';
import { LogoutButton } from '@/components/LogoutButton';
import ThemeSwitcher from '@/components/ThemeSwitcher';
import LanguageSwitcher from '@/components/LanguageSwitcher';
import { Button } from '@/components/ui/button';
import { Settings, Shield } from 'lucide-react';
import { MobileMenu } from '@/components/MobileMenu';

interface LayoutProps {
  showSimpleHeader?: boolean;
}

export function Layout({ showSimpleHeader = false }: LayoutProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { isAuthenticated, multiUserMode, user } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleGoHome = () => {
    navigate('/');
  };

  const handleGoBack = () => {
    navigate(-1);
  };

  const handleNavigateToSettings = () => {
    navigate('/settings');
  };

  const handleNavigateToAdmin = () => {
    navigate('/admin');
  };

  // Determine which type of header to show based on route
  const isSimpleHeaderRoute = ['/reset-password', '/register', '/settings', '/admin'].some(
    route => location.pathname.startsWith(route)
  );

  const shouldShowSimpleHeader = showSimpleHeader || isSimpleHeaderRoute;

  if (shouldShowSimpleHeader) {
    return (
      <>
        <header className="flex items-center justify-between px-8 py-2 bg-background">
          <button onClick={handleGoBack} className="cursor-pointer">
            <Logo className="h-16 w-32 text-foreground dark:text-foreground-dark" />
          </button>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <ThemeSwitcher size={24} />
          </div>
        </header>
        <main>
          <Outlet />
        </main>
      </>
    );
  }

  return (
    <>
      <header className="flex items-center justify-between px-8 py-2 bg-background">
        <button onClick={handleGoHome} className="cursor-pointer">
          <Logo className="h-16 w-32 text-foreground dark:text-foreground-dark" />
        </button>
        <div className="hidden sm:flex items-center gap-4">
          {isAuthenticated && multiUserMode && (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleNavigateToSettings}
                className="flex items-center gap-2"
              >
                <Settings className="h-4 w-4" />
                <span className="sm:hidden lg:inline">{t("app.settings")}</span>
              </Button>

              {user?.is_admin && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleNavigateToAdmin}
                  className="flex items-center gap-2"
                >
                  <Shield className="h-4 w-4" />
                  <span className="sm:hidden lg:inline">{t("app.admin")}</span>
                </Button>
              )}
            </>
          )}
          <LogoutButton />
          <LanguageSwitcher />
          <ThemeSwitcher size={24} />
        </div>
        <div className="sm:hidden">
          <MobileMenu
            showSettings={isAuthenticated && multiUserMode}
            showAdmin={isAuthenticated && multiUserMode && !!user?.is_admin}
            onOpenSettings={handleNavigateToSettings}
            onOpenAdmin={handleNavigateToAdmin}
            isOpen={mobileMenuOpen}
            onOpenChange={setMobileMenuOpen}
          />
        </div>
      </header>
      <main>
        <Outlet />
      </main>
    </>
  );
}