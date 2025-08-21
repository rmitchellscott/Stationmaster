import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '@/components/AuthProvider';
import { LoginForm } from '@/components/LoginForm';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requireAuth?: boolean;
  requireAdmin?: boolean;
}

export function ProtectedRoute({ 
  children, 
  requireAuth = true, 
  requireAdmin = false 
}: ProtectedRouteProps) {
  const { isAuthenticated, isLoading, login, authConfigured, user } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return null;
  }

  // If authentication is required but user is not authenticated
  if (requireAuth && authConfigured && !isAuthenticated) {
    return <LoginForm onLogin={login} />;
  }

  // If admin access is required but user is not admin
  if (requireAdmin && isAuthenticated && !user?.is_admin) {
    return <Navigate to="/" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}