import React, { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, useNavigate } from 'react-router-dom';
import HomePage from './HomePage';
import { ThemeProvider } from '@/components/theme-provider';
import { AuthProvider } from '@/components/AuthProvider';
import { ConfigProvider } from '@/components/ConfigProvider';
import { Layout } from '@/components/Layout';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { UserSettingsPage } from '@/components/UserSettingsPage';
import { PrivatePluginEditorPage } from '@/components/PrivatePluginEditorPage';
import { AddPluginPage } from '@/components/AddPluginPage';
import { AdminPage } from '@/components/AdminPage';
import { PasswordReset } from '@/components/PasswordReset';
import { RegisterForm } from '@/components/RegisterForm';
import { oauthService } from '@/services/oauthService';
import { toast, Toaster } from 'sonner';

function AppContent() {
  const navigate = useNavigate();

  useEffect(() => {
    // Handle OAuth return flow
    const handleOAuthReturn = () => {
      const result = oauthService.handleOAuthReturn();
      
      if (result.success && result.provider) {
        const providerName = oauthService.getProviderDisplayName(result.provider);
        toast.success(`Successfully connected to ${providerName}!`);
        
        // Redirect to stored return URL or home
        const returnUrl = oauthService.getReturnUrl();
        if (returnUrl && returnUrl !== '/') {
          navigate(returnUrl);
        }
      } else if (result.provider && result.error) {
        const providerName = oauthService.getProviderDisplayName(result.provider);
        toast.error(`Failed to connect to ${providerName}: ${result.error}`);
        
        // Still redirect to return URL so user can retry
        const returnUrl = oauthService.getReturnUrl();
        if (returnUrl && returnUrl !== '/') {
          navigate(returnUrl);
        }
      }
    };

    handleOAuthReturn();
  }, [navigate]);
  
  return (
    <>
      <Routes>
        {/* Public routes */}
        <Route path="/reset-password" element={<PasswordReset />} />
        <Route path="/register" element={<RegisterForm />} />
        
        {/* Protected routes */}
        <Route path="/" element={<Layout />}>
          <Route index element={
            <ProtectedRoute>
              <HomePage />
            </ProtectedRoute>
          } />
          <Route path="settings" element={
            <ProtectedRoute>
              <UserSettingsPage />
            </ProtectedRoute>
          } />
          <Route path="plugins/private/edit" element={
            <ProtectedRoute>
              <PrivatePluginEditorPage />
            </ProtectedRoute>
          } />
          <Route path="plugins/add" element={
            <ProtectedRoute>
              <AddPluginPage />
            </ProtectedRoute>
          } />
          <Route path="admin" element={
            <ProtectedRoute requireAdmin>
              <AdminPage />
            </ProtectedRoute>
          } />
        </Route>
      </Routes>
    </>
  );
}

export default function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <ConfigProvider>
        <BrowserRouter>
          <AuthProvider>
            <AppContent />
            <Toaster richColors position="top-right" />
          </AuthProvider>
        </BrowserRouter>
      </ConfigProvider>
    </ThemeProvider>
  );
}