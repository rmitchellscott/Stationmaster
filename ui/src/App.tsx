import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import HomePage from './HomePage';
import { ThemeProvider } from '@/components/theme-provider';
import ThemeSwitcher from '@/components/ThemeSwitcher';
import LanguageSwitcher from '@/components/LanguageSwitcher';
import { Logo } from '@/components/Logo';
import { AuthProvider, useAuth } from '@/components/AuthProvider';
import { ConfigProvider } from '@/components/ConfigProvider';
import { LogoutButton } from '@/components/LogoutButton';
import { UserSettingsPage } from '@/components/UserSettingsPage';
import { AdminPage } from '@/components/AdminPage';
import { AdminPanel } from '@/components/AdminPanel';
import { PasswordReset } from '@/components/PasswordReset';
import { RegisterForm } from '@/components/RegisterForm';
import { Button } from '@/components/ui/button';
import { Settings, Shield } from 'lucide-react';
import { MobileMenu } from '@/components/MobileMenu';

function AppContent() {
  const { t } = useTranslation();
  const { isAuthenticated, multiUserMode, user } = useAuth();
  const [showAdminPanel, setShowAdminPanel] = useState(false);
  
  const isPasswordResetPage = window.location.pathname === '/reset-password' || window.location.search.includes('token=');
  const isRegistrationPage = window.location.pathname === '/register';
  const isSettingsPage = window.location.pathname === '/settings';
  const isAdminPage = window.location.pathname === '/admin';

  if (isPasswordResetPage) {
    return (
      <>
        <header className="flex items-center justify-between px-8 py-2 bg-background">
        <button onClick={() => window.location.href = '/'} className="cursor-pointer">
          <Logo className="h-16 w-32 text-foreground dark:text-foreground-dark" />
        </button>
        <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <ThemeSwitcher size={24} />
          </div>
        </header>
        <main>
          <PasswordReset onBack={() => window.location.href = '/'} />
        </main>
      </>
    );
  }
  
  if (isRegistrationPage) {
    return (
      <>
        <header className="flex items-center justify-between px-8 py-2 bg-background">
        <button onClick={() => window.location.href = '/'} className="cursor-pointer">
          <Logo className="h-16 w-32 text-foreground dark:text-foreground-dark" />
        </button>
        <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <ThemeSwitcher size={24} />
          </div>
        </header>
        <main>
          <RegisterForm />
        </main>
      </>
    );
  }

  if (isSettingsPage) {
    const handleNavigateBack = () => {
      window.history.back();
    };

    return (
      <>
        <header className="flex items-center justify-between px-8 py-2 bg-background">
          <button onClick={handleNavigateBack} className="cursor-pointer">
            <Logo className="h-16 w-32 text-foreground dark:text-foreground-dark" />
          </button>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <ThemeSwitcher size={24} />
          </div>
        </header>
        <main>
          <UserSettingsPage onNavigateBack={handleNavigateBack} />
        </main>
      </>
    );
  }

  if (isAdminPage) {
    const handleNavigateBack = () => {
      window.history.back();
    };

    return (
      <>
        <header className="flex items-center justify-between px-8 py-2 bg-background">
          <button onClick={handleNavigateBack} className="cursor-pointer">
            <Logo className="h-16 w-32 text-foreground dark:text-foreground-dark" />
          </button>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <ThemeSwitcher size={24} />
          </div>
        </header>
        <main>
          <AdminPage onNavigateBack={handleNavigateBack} />
        </main>
      </>
    );
  }

  return (
    <>
      <header className="flex items-center justify-between px-8 py-2 bg-background">
        <Logo className="h-16 w-32 text-foreground dark:text-foreground-dark" />
        <div className="hidden sm:flex items-center gap-4">
          {isAuthenticated && multiUserMode && (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => window.location.href = '/settings'}
                className="flex items-center gap-2"
              >
                <Settings className="h-4 w-4" />
                <span className="sm:hidden lg:inline">{t("app.settings")}</span>
              </Button>

              {user?.is_admin && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => window.location.href = '/admin'}
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
            onOpenSettings={() => window.location.href = '/settings'}
            onOpenAdmin={() => window.location.href = '/admin'}
          />
        </div>
      </header>
      <main>
        <HomePage />
      </main>
      
      {/* Modals */}
      <AdminPanel 
        isOpen={showAdminPanel} 
        onClose={() => setShowAdminPanel(false)} 
      />
    </>
  );
}

export default function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <ConfigProvider>
        <AuthProvider>
          <AppContent />
        </AuthProvider>
      </ConfigProvider>
    </ThemeProvider>
  );
}