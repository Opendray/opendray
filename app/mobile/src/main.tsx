import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import './index.css'
import '@/stores/theme' // side-effect: apply persisted theme
import { App } from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
