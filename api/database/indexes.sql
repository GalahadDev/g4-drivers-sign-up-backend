-- Optimización de índices para G4 Drivers App

-- 1. Índice para búsqueda rápida de referidos (CRÍTICO para el Dashboard)
-- Permite filtrar por referred_by_code instantáneamente.
CREATE INDEX IF NOT EXISTS idx_profiles_referred_by_code ON public.profiles(referred_by_code);

-- 2. Índice para ordenamiento por fecha (Mejora ORDER BY created_at DESC)
CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON public.profiles(created_at DESC);

-- 3. Índice para filtrar aplicaciones por estado (Útil para Admin Panel "Pending" vs "Approved")
CREATE INDEX IF NOT EXISTS idx_driver_applications_status ON public.driver_applications(status);

-- 4. Índice compuesto para búsqueda de usuario por email (aunque suele ser unique, ayuda en búsquedas parciales si se usa LIKE)
CREATE INDEX IF NOT EXISTS idx_profiles_email ON public.profiles(email);

-- 5. Índice para buscar aplicaciones por user_id (JOINs más rápidos)
-- Aunque es FK y suele tener índice implícito en algunos sistemas, mejor asegurar.
CREATE INDEX IF NOT EXISTS idx_driver_applications_user_id ON public.driver_applications(user_id);
