import React, { useState } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import HomePage from './HomePage';
import { ThemeProvider } from '@/components/theme-provider';
import { AuthProvider } from '@/components/AuthProvider';
import { ConfigProvider } from '@/components/ConfigProvider';
import { Layout } from '@/components/Layout';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { UserSettingsPage } from '@/components/UserSettingsPage';
import { AdminPage } from '@/components/AdminPage';
import { AdminPanel } from '@/components/AdminPanel';
import { PasswordReset } from '@/components/PasswordReset';
import { RegisterForm } from '@/components/RegisterForm';

function AppContent() {
  const [showAdminPanel, setShowAdminPanel] = useState(false);
  
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
          <Route path="admin" element={
            <ProtectedRoute requireAdmin>
              <AdminPage />
            </ProtectedRoute>
          } />
        </Route>
      </Routes>
      
      {/* Global modals */}
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
        <BrowserRouter>
          <AuthProvider>
            <AppContent />
          </AuthProvider>
        </BrowserRouter>
      </ConfigProvider>
    </ThemeProvider>
  );
}