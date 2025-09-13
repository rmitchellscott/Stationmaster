'use client'

import { createContext, useContext, useEffect, useState, useCallback, useRef, ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { useConfig } from './ConfigProvider'

const AUTH_CHECK_EXPIRY_MS = 5 * 60 * 1000; // 5 minutes

// Helper to get appropriate storage (sessionStorage for auth isolation)
const getAuthStorage = () => {
  if (typeof window === 'undefined') return null
  return window.sessionStorage || window.localStorage
}

interface User {
  id: string
  username: string
  email: string
  is_admin: boolean
  created_at: string
  last_login?: string
  onboarding_completed?: boolean
  timezone?: string
}

interface AuthContextType {
  isAuthenticated: boolean
  isLoading: boolean
  authConfigured: boolean
  multiUserMode: boolean
  uiSecret: string | null
  user: User | null
  oidcEnabled: boolean
  proxyAuthEnabled: boolean
  login: () => Promise<void>
  logout: () => void
  refetchAuth: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: ReactNode
}

export function AuthProvider({ children }: AuthProviderProps) {
  const location = useLocation();
  const isPublicRoute = ['/reset-password', '/register'].includes(location.pathname);
  
  // Track route transitions to detect public → protected transitions
  const [prevRoute, setPrevRoute] = useState<string | null>(null);
  const [wasOnPublicRoute, setWasOnPublicRoute] = useState<boolean>(false);
  
  
  const authStorage = getAuthStorage()
  const storedConf = authStorage?.getItem('authConfigured') || null
  const initialAuthConfigured = storedConf === 'true'
  const [authConfigured, setAuthConfigured] = useState<boolean>(initialAuthConfigured)
  const [multiUserMode, setMultiUserMode] = useState<boolean>(false)
  const [oidcEnabled, setOidcEnabled] = useState<boolean>(false)
  const [proxyAuthEnabled, setProxyAuthEnabled] = useState<boolean>(false)
  const [uiSecret, setUiSecret] = useState<string | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const { config: configData } = useConfig()
  
  // Add request deduplication for auth checks
  const authPromise = useRef<Promise<void> | null>(null)
  // Track when auth has been explicitly cleared to prevent re-authentication
  const explicitlyCleared = useRef<boolean>(false)
  
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(() => {
    if (typeof window !== 'undefined') {
      // Check if we have a UI secret injected (means web auth is disabled)
      const hasUISecret = !!(window as Window & { __UI_SECRET__?: string }).__UI_SECRET__
      if (hasUISecret) {
        // UI secret present - we'll need to call server to get JWT, start as false
        return false
      }
      
      const storage = getAuthStorage()
      const authConfigured = storage?.getItem('authConfigured')
      if (authConfigured === 'false') {
        // No web auth configured and no UI secret, start as authenticated
        return true
      }
      
      // Web auth is configured - be more conservative about initial state
      // Only trust storage if we recently checked
      const expiry = parseInt(storage?.getItem('authExpiry') || '0', 10)
      const lastCheck = parseInt(storage?.getItem('lastAuthCheck') || '0', 10)
      const recentCheck = Date.now() - lastCheck < AUTH_CHECK_EXPIRY_MS
      
      return recentCheck && expiry > Date.now()
    }
    return false
  })
  const [isLoading, setIsLoading] = useState<boolean>(true) // Always start loading

  const checkAuth = useCallback(async () => {
    if (!configData) return
    
    // If an auth check is already in progress, return the existing promise
    if (authPromise.current) {
      return authPromise.current
    }

    const promise = (async () => {
      try {
      
      // Check for proxy auth first (multi-user mode only)
      if (configData.multiUserMode && configData.proxyAuthEnabled) {
        // Try proxy auth endpoint
        const proxyResponse = await fetch('/api/auth/proxy/check', {
          credentials: 'include'
        })
        const proxyData = await proxyResponse.json()
        
        if (proxyData.authenticated) {
          // Proxy auth successful
          setAuthConfigured(false) // No manual login needed
          setIsAuthenticated(true)
          setUser(proxyData.user || null)
          // Set the config data too
          setMultiUserMode(configData.multiUserMode || false)
          setOidcEnabled(configData.oidcEnabled || false)
          setProxyAuthEnabled(configData.proxyAuthEnabled || false)
          if (typeof window !== 'undefined') {
            const storage = getAuthStorage()
            storage?.setItem('authConfigured', 'false')
            const expiry = Date.now() + 24 * 3600 * 1000
            storage?.setItem('authExpiry', expiry.toString())
            storage?.setItem('lastAuthCheck', Date.now().toString())
          }
          return
        } else if (proxyData.proxy_available === false) {
          // Proxy auth is enabled but header not present - fall through to other auth methods
          // Continue to regular auth flow below
        } else {
          // Proxy auth failed for other reasons (user not found, inactive, etc)
          setAuthConfigured(true) // Show login error
          setIsAuthenticated(false)
          setUser(null)
          return
        }
      }
      
      // Get UI secret from window (injected by server when web auth is disabled)
      const uiSecret = (window as Window & { __UI_SECRET__?: string }).__UI_SECRET__ || null
      setUiSecret(uiSecret)
      
      if (configData.authEnabled) {
        // Web authentication is enabled - users need to log in
        setAuthConfigured(true)
        if (typeof window !== 'undefined') {
          const storage = getAuthStorage()
          storage?.setItem('authConfigured', 'true')
        }
        const response = await fetch('/api/auth/check', {
          credentials: 'include',
          headers: {
            'X-UI-Token': uiSecret || ''
          }
        })
        const data = await response.json()
        setIsAuthenticated(data.authenticated)
        setUser(data.user || null)
        if (typeof window !== 'undefined') {
          const storage = getAuthStorage()
          if (data.authenticated) {
            const expiry = Date.now() + 24 * 3600 * 1000
            storage?.setItem('authExpiry', expiry.toString())
          } else {
            storage?.setItem('authExpiry', '0')
          }
        }
      } else if (uiSecret) {
        // Web auth is disabled but we have UI secret - call auth check to get auto-JWT
        setAuthConfigured(false)
        if (typeof window !== 'undefined') {
          const storage = getAuthStorage()
          storage?.setItem('authConfigured', 'false')
        }
        const response = await fetch('/api/auth/check', {
          credentials: 'include',
          headers: {
            'X-UI-Token': uiSecret
          }
        })
        const data = await response.json()
        setIsAuthenticated(data.authenticated)
        setUser(data.user || null)
        if (typeof window !== 'undefined') {
          const storage = getAuthStorage()
          const expiry = Date.now() + 24 * 3600 * 1000
          storage?.setItem('authExpiry', expiry.toString())
        }
      } else if (configData.apiKeyEnabled) {
        // Only API key auth is enabled (for external clients)
        // UI users are automatically authenticated
        setAuthConfigured(false)
        setIsAuthenticated(true)
        if (typeof window !== 'undefined') {
          const storage = getAuthStorage()
          storage?.setItem('authConfigured', 'false')
          const expiry = Date.now() + 365 * 24 * 3600 * 1000
          storage?.setItem('authExpiry', expiry.toString())
        }
      } else {
        // No authentication configured at all
        setAuthConfigured(false)
        setIsAuthenticated(true)
        if (typeof window !== 'undefined') {
          const storage = getAuthStorage()
          storage?.setItem('authConfigured', 'false')
          const expiry = Date.now() + 365 * 24 * 3600 * 1000
          storage?.setItem('authExpiry', expiry.toString())
        }
      }
    } catch {
      // Default to authenticated if we can't check (fail open for UI)
      setAuthConfigured(false)
      setIsAuthenticated(true)
      if (typeof window !== 'undefined') {
        const storage = getAuthStorage()
        storage?.setItem('authConfigured', 'false')
        const expiry = Date.now() + 365 * 24 * 3600 * 1000
        storage?.setItem('authExpiry', expiry.toString())
      }
    } finally {
      setIsLoading(false)
      if (typeof window !== 'undefined') {
        document.documentElement.classList.remove('auth-check')
      }
      authPromise.current = null // Clear promise after completion
    }
    })()

    authPromise.current = promise
    return promise
  }, [configData])

  const login = async () => {
    // Don't set isAuthenticated immediately
    // Instead, re-check auth status to ensure JWT cookie is set
    await checkAuth()
  }

  const logout = async () => {
    if (!authConfigured) return
    
    try {
      await fetch('/api/auth/logout', { 
        method: 'POST',
        credentials: 'include'
      })
    } catch {
      // Handle error silently
    }
    
    // Clear all auth-related state and localStorage
    setIsAuthenticated(false)
    setUser(null)
    setUiSecret(null)
    
    if (typeof window !== 'undefined') {
      const storage = getAuthStorage()
      storage?.setItem('authExpiry', '0')
      storage?.setItem('lastAuthCheck', '0')
      // Trigger a custom event to notify other components
      window.dispatchEvent(new CustomEvent('logout'))
    }
  }

  // Track route transitions
  useEffect(() => {
    if (prevRoute !== null) { // Skip initial load
      const wasPublic = ['/reset-password', '/register'].includes(prevRoute);
      const isPublicToProtected = wasPublic && !isPublicRoute;
      
      if (isPublicToProtected) {
        // Clear any potentially corrupted auth state
        authPromise.current = null;
        setIsLoading(true);
        setIsAuthenticated(false); // Force unauthenticated state initially
        setUser(null);
        
        if (explicitlyCleared.current) {
          // Auth was explicitly cleared - force login requirement without checkAuth
          setAuthConfigured(true);
          setIsLoading(false);
          explicitlyCleared.current = false; // Reset flag
          if (typeof window !== 'undefined') {
            const storage = getAuthStorage()
            storage?.setItem('authConfigured', 'true');
            storage?.setItem('authExpiry', '0');
            storage?.setItem('lastAuthCheck', '0');
          }
        } else {
          // Normal transition - set up auth and check
          setAuthConfigured(true); // Ensure auth is required to show login form
          // Clear storage auth state
          if (typeof window !== 'undefined') {
            const storage = getAuthStorage()
            storage?.setItem('authExpiry', '0');
            storage?.setItem('lastAuthCheck', '0');
            storage?.setItem('authConfigured', 'true'); // Force auth required state
          }
          // Force fresh auth check
          if (configData) {
            checkAuth();
          }
        }
      }
      
      // When navigating TO a public route, fully logout to prevent stale data
      if (!wasPublic && isPublicRoute) {
        // Mark as explicitly cleared to prevent re-authentication
        explicitlyCleared.current = true;
        
        // Call logout endpoint to invalidate backend session/JWT
        if (authConfigured) {
          try {
            fetch('/api/auth/logout', { 
              method: 'POST',
              credentials: 'include'
            }).catch(() => {
              // Handle error silently - we're clearing state anyway
            });
          } catch {
            // Handle error silently
          }
        }
        
        // Clear all frontend state
        setIsAuthenticated(false);
        setUser(null);
        setAuthConfigured(false);
        setUiSecret(null);
        
        // Clear all storage auth data
        if (typeof window !== 'undefined') {
          const storage = getAuthStorage()
          storage?.setItem('authExpiry', '0');
          storage?.setItem('lastAuthCheck', '0');
          storage?.removeItem('authConfigured');
        }
      }
    }
    
    // Update tracking state
    setPrevRoute(location.pathname);
    setWasOnPublicRoute(isPublicRoute);
  }, [location.pathname, isPublicRoute, configData]);

  useEffect(() => {
    if (configData && !isPublicRoute) {
      // Set multi-user mode and auth methods from config
      setMultiUserMode(configData.multiUserMode || false)
      setOidcEnabled(configData.oidcEnabled || false)
      setProxyAuthEnabled(configData.proxyAuthEnabled || false)
      
      // Only run auth check if we haven't already done it via route transition
      if (!authPromise.current) {
        checkAuth()
      }
    } else if (isPublicRoute) {
      // For public routes, we should also remove the auth-check class and stop loading
      setIsLoading(false)
      if (typeof window !== 'undefined') {
        document.documentElement.classList.remove('auth-check')
      }
    }
  }, [configData, isPublicRoute])

  useEffect(() => {
    if (!isLoading && typeof window !== 'undefined') {
      document.documentElement.classList.remove('auth-check')
    }
  }, [isLoading])

  return (
    <AuthContext.Provider value={{ isAuthenticated, isLoading, login, logout, authConfigured, multiUserMode, uiSecret, user, oidcEnabled, proxyAuthEnabled, refetchAuth: checkAuth }}>
      {children}
    </AuthContext.Provider>
  )
}
