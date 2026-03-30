import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.planner.app',
  appName: 'Planner',
  webDir: 'dist',
  server: {
    // In native builds, VITE_API_BASE_URL must be set to the public API domain
    // so fetch() calls resolve to an absolute URL instead of the local filesystem.
    androidScheme: 'https',
    iosScheme: 'https',
  },
  plugins: {
    Camera: {
      // iOS permission descriptions (required by App Store)
      presentationStyle: 'fullscreen',
    },
  },
}

export default config
